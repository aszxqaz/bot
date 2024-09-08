package payeer

import (
	"log/slog"

	"github.com/shopspring/decimal"
)

var cent = decimal.RequireFromString(".01")

type PayeerPriceSelectorContext struct {
	info   *PairsOrderInfo
	action Action
}

type PayeerPriceSelectorConfig struct {
	PlacementValueOffset   decimal.Decimal
	ElevationPriceFraction decimal.Decimal
	MaxWmaSurplus          decimal.Decimal
	WmaTake                int
	WmaTakeAmount          decimal.Decimal
}

type PayeerPriceSelector struct {
	config *PayeerPriceSelectorConfig
}

func NewPayeerPriceSelector(config *PayeerPriceSelectorConfig) *PayeerPriceSelector {
	return &PayeerPriceSelector{
		config: config,
	}
}

func (ps *PayeerPriceSelector) SelectPrice(action Action, info *PairsOrderInfo) (bool, decimal.Decimal) {
	pctx := &PayeerPriceSelectorContext{
		info:   info,
		action: action,
	}
	return ps.pipe(
		pctx,
		ps.selectByValueOffset,
		ps.selectByElevation,
		ps.filterByWmaRatio,
	)
}

func (ps *PayeerPriceSelector) filterByWmaRatio(pctx *PayeerPriceSelectorContext, prevPrice decimal.Decimal) (bool, decimal.Decimal) {
	orders := ps.resolveOrders(pctx)
	wma := ps.getWeightedMeanAverage(orders)
	isOk := false
	var wmaVal decimal.Decimal
	if pctx.action == ACTION_SELL {
		wmaVal = decimal.NewFromInt(1).Sub(ps.config.MaxWmaSurplus).Mul(wma)
		isOk = prevPrice.GreaterThan(wmaVal)
	} else {
		wmaVal = decimal.NewFromInt(1).Add(ps.config.MaxWmaSurplus).Mul(wma)
		isOk = prevPrice.LessThan(wmaVal)
	}
	slog.Info("[PayeerPriceSelector] filtered by wma ratio", "ok", isOk, "action", pctx.action, "price", prevPrice.String(), "wma", wma.StringFixed(2), "wma surplus", ps.config.MaxWmaSurplus.StringFixed(6), "wma adjusted", wmaVal.StringFixed(2))
	return isOk, prevPrice
}

func (ps *PayeerPriceSelector) selectByElevation(pctx *PayeerPriceSelectorContext, prevPrice decimal.Decimal) (bool, decimal.Decimal) {
	fractionAbs := ps.config.ElevationPriceFraction.Mul(prevPrice)
	orders := ps.resolveOrders(pctx)
	prevPriceIndex := 0
	for i, order := range orders {
		orderPrice := decimal.RequireFromString(order.Price)
		hasFound := prevPrice.GreaterThanOrEqual(orderPrice)
		if pctx.action == ACTION_SELL {
			hasFound = prevPrice.LessThanOrEqual(orderPrice)
		}
		if hasFound {
			prevPriceIndex = i
			break
		}
	}
	afterPrice := prevPrice.Copy()
	for i := prevPriceIndex; i >= 0; i-- {
		price := decimal.RequireFromString(orders[i].Price)
		diff := price.Sub(afterPrice)
		if pctx.action == ACTION_SELL {
			diff = afterPrice.Sub(price)
		}
		if diff.LessThanOrEqual(fractionAbs) && diff.GreaterThanOrEqual(decimal.Zero) {
			afterPrice = ps.elevatePrice(pctx, afterPrice, diff.Add(cent))
			fractionAbs = fractionAbs.Sub(diff.Add(cent))
			if fractionAbs.LessThanOrEqual(decimal.Zero) {
				break
			}
		}
	}
	slog.Info("[PayeerPriceSelector] selected by elevation", "price", afterPrice.StringFixed(2), "fractionAbs", fractionAbs.StringFixed(2), "elevation", afterPrice.Sub(prevPrice).Abs())
	return true, afterPrice
}

func (ps *PayeerPriceSelector) selectByValueOffset(pctx *PayeerPriceSelectorContext, prevPrice decimal.Decimal) (bool, decimal.Decimal) {
	acc := decimal.NewFromInt(0)
	var selectedPrice decimal.Decimal
	orders := ps.resolveOrders(pctx)
	for _, order := range orders {
		value, _ := decimal.NewFromString(order.Value)
		acc = acc.Add(value)
		if acc.GreaterThanOrEqual(ps.config.PlacementValueOffset) {
			price, err := decimal.NewFromString(order.Price)
			if err != nil {
				panic(err)
			}
			selectedPrice = ps.elevatePrice(pctx, price, cent)
			break
		}
	}
	slog.Info("[PayeerPriceSelector] selected by value offset", "price", selectedPrice.StringFixed(2))
	return true, selectedPrice
}

func (ps *PayeerPriceSelector) elevatePrice(pctx *PayeerPriceSelectorContext, price decimal.Decimal, diff decimal.Decimal) decimal.Decimal {
	if pctx.action == ACTION_SELL {
		return price.Sub(diff)
	} else {
		return price.Add(diff)
	}
}

func (ps *PayeerPriceSelector) resolveOrders(pctx *PayeerPriceSelectorContext) []OrdersOrder {
	if pctx.action == ACTION_SELL {
		return pctx.info.Asks
	} else {
		return pctx.info.Bids
	}
}

type PayeerPipeFn = func(pctx *PayeerPriceSelectorContext, prevPrice decimal.Decimal) (bool, decimal.Decimal)

func (ps *PayeerPriceSelector) pipe(
	pctx *PayeerPriceSelectorContext,
	fns ...PayeerPipeFn,
) (bool, decimal.Decimal) {
	prevPrice := decimal.Zero
	for _, fn := range fns {
		ok, price := fn(pctx, prevPrice)
		if !ok {
			return false, price
		}
		prevPrice = price
	}
	return true, prevPrice
}

func (ps *PayeerPriceSelector) getWeightedMeanAverage(orders []OrdersOrder) decimal.Decimal {
	totalValue := decimal.NewFromInt(0)
	totalAmount := decimal.NewFromInt(0)
	for i, order := range orders {
		value, _ := decimal.NewFromString(order.Value)
		amount, _ := decimal.NewFromString(order.Amount)
		totalValue = totalValue.Add(value)
		totalAmount = totalAmount.Add(amount)
		if (ps.config.WmaTake > 0 && i == ps.config.WmaTake) ||
			(ps.config.WmaTakeAmount.IsPositive() && totalAmount.GreaterThan(ps.config.WmaTakeAmount)) {
			break
		}
	}
	return totalValue.Div(totalAmount)
}
