package mexc

import (
	"log"
	httpclient "mexc-bot/http_client"
	"sync"
	"time"
)

type listenKeyManager struct {
	listenKey  string
	time       time.Time
	mu         sync.Mutex
	httpClient *httpclient.HttpClient
	qm         *queryMaker
}

func (l *listenKeyManager) setListenKey() error {
	listenKey, err := l.getListenKey()
	if err != nil {
		log.Fatal("failed to post listen key", err)
	}
	if listenKey == "" {
		listenKey, err = l.postListenKey()
		if err != nil {
			log.Fatal("failed to post listen key", err)
		}
	} else {
		err = l.putListenKey(listenKey)
		if err != nil {
			log.Fatal("failed to put listen key", err)
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.listenKey = listenKey
	return nil
}

func (l *listenKeyManager) Start() {
	l.setListenKey()
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			l.setListenKey()
		}
	}()
}

func (l *listenKeyManager) ListenKey() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.listenKey
}

func (l *listenKeyManager) getListenKey() (string, error) {
	var response struct {
		ListenKey []string `json:"listenKey"`
	}
	err := l.httpClient.Get("/userDataStream?"+l.qm.defaultSignature(), &response)
	if err != nil {
		log.Println("[ListenKeyManager] Failed to get listen keys: ", response.ListenKey)
		return "", err
	}
	if len(response.ListenKey) == 0 {
		return "", nil
	}
	return response.ListenKey[0], nil
}

func (l *listenKeyManager) postListenKey() (string, error) {
	var response struct {
		ListenKey string `json:"listenKey"`
	}
	err := l.httpClient.Post("/userDataStream?"+l.qm.defaultSignature(), &response)
	if err != nil {
		log.Println("[ListenKeyManager] Failed to create listen key: ", response.ListenKey)
		return "", err
	}
	log.Println("[ListenKeyManager] Listen key created: ", response.ListenKey)
	return response.ListenKey, nil
}

func (l *listenKeyManager) putListenKey(key string) error {
	var response struct {
		ListenKey string `json:"listenKey"`
	}
	err := l.httpClient.Put("/userDataStream?"+l.qm.getListenKeyQuery(key), &response)
	if err != nil {
		log.Println("[ListenKeyManager] Failed to put listen key: ", err)
		return err
	}
	log.Println("[ListenKeyManager] Listen key made keep-alive: ", response.ListenKey)
	return nil
}
