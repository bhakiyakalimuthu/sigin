/*
Copyright © 2022
Author Bhakiyaraj Kalimuthu
Email bhakiya.kalimuthu@gmail.com
*/

package internal

import (
	"context"
	"fmt"
	"sigin/store"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	_NumberOfPages = 1
)

// Sigin is the interface that groups the Start and Process & create methods
type Sigin interface {
	Process(wg *sync.WaitGroup, workerID int)
	Start(ctx context.Context)
	CreateJobs()
}

// sigin type
type sigin struct {
	logger        *zap.Logger // logger
	httpClient    HttpClient  // http client for getting method signature
	db            store.Store
	interval      time.Duration  // interval in which job needs to be processed
	producerChan  chan *Response // channel to process jobs
	consumerChan  chan *Response // chanel to consume jobs
	serverAddress string
}

// NewSigin constructor
func NewSigin(logger *zap.Logger, httpClient HttpClient, db store.Store, interval time.Duration, producerChan, consumerChan chan *Response, serverAddress string) Sigin {
	return &sigin{
		logger:        logger,
		interval:      interval,
		producerChan:  producerChan,
		consumerChan:  consumerChan,
		httpClient:    httpClient,
		db:            db,
		serverAddress: serverAddress,
	}
}

// Process starts the worker process based on the number items in the consumer channel until it closes
func (s *sigin) Process(wg *sync.WaitGroup, workerID int) {
	defer wg.Done()
	for job := range s.consumerChan {
		//<-time.After(s.interval) // wait for the provided interval
		s.logger.Debug("starting job", zap.Int("workerID", workerID))
		entry := s.getMethodSignatures(job)
		if entry != nil {
			if err := s.db.Insert(entry); err != nil {
				s.logger.Error("failed to insert", zap.Error(err))
			}
		}

	}
	s.logger.Warn("gracefully finishing job", zap.Int("workerID", workerID))
}

// Start acts as a proxy between producer and consumer channel,also supports the graceful cancellation
func (s *sigin) Start(ctx context.Context) {
	for {
		select {
		case job := <-s.producerChan: // fetch job from producer
			s.logger.Debug("received msg from consumerChan")
			s.consumerChan <- job // pass job to consumer
		case <-ctx.Done():
			s.logger.Warn("received context cancellation......")
			close(s.consumerChan) // when context is done, close the consumer channel
			return
		}
	}
}

func (s *sigin) CreateJobs() {
	// https://www.4byte.directory/api/v1/signatures/?page={page_number}
	tryAgain := make([]int, 0) // store failed pages to try again
	for pageNum := 1; pageNum <= _NumberOfPages; pageNum++ {
		pageURL := fmt.Sprintf("%s/?page=%d", s.serverAddress, pageNum)
		resp, err := s.httpClient.GetMethodSignature(pageURL)
		if err != nil {
			tryAgain = append(tryAgain, pageNum)
			continue
		}
		s.logger.Info("pushing jobs to producer......", zap.Int("pageNum", pageNum))
		s.producerChan <- resp
	}
	s.logger.Warn("tryAgain failed pages", zap.Ints("pageNums", tryAgain))
	for _, pageNum := range tryAgain {
		pageURL := fmt.Sprintf("%s/?page=%d", s.serverAddress, pageNum)
		resp, _ := s.httpClient.GetMethodSignature(pageURL)
		if resp != nil {
			s.logger.Info("pushing jobs to producer......", zap.Int("pageNum", pageNum))
			s.producerChan <- resp
		}
	}

}

func (s *sigin) getMethodSignatures(in *Response) []*store.MethodSignatureEntry {
	if in == nil {
		s.logger.Debug("empty response")
		return nil
	}
	out := make([]*store.MethodSignatureEntry, 0, len(in.Results))
	for _, entry := range in.Results {
		out = append(out, &store.MethodSignatureEntry{
			Id:            entry.ID,
			TextSignature: entry.TextSignature,
			HexSignature:  entry.HexSignature,
		})
	}
	return out
}

type Response struct {
	Count    int         `json:"count"`
	Next     string      `json:"next"`
	Previous interface{} `json:"previous"`
	Results  []struct {
		ID             int       `json:"id"`
		CreatedAt      time.Time `json:"created_at"`
		TextSignature  string    `json:"text_signature"`
		HexSignature   string    `json:"hex_signature"`
		BytesSignature string    `json:"bytes_signature"`
	} `json:"results"`
}
