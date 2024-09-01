package main

type BooksData struct {
	Asks []Book `json:"asks"`
	Bids []Book `json:"bids"`
}

type Book struct {
	Price    string `json:"p"`
	Quantity string `json:"v"`
}

type BooksResponse struct {
	Endpoint string    `json:"c"`
	Symbol   string    `json:"s"`
	Time     int       `json:"t"`
	Data     BooksData `json:"d"`
}
