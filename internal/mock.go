/*
Copyright © 2022
Author Bhakiyaraj Kalimuthu
Email bhakiya.kalimuthu@gmail.com
*/

package internal

import (
	"github.com/stretchr/testify/mock"
)

type httpClientMock struct {
	mock.Mock
}

func (n *httpClientMock) GetMethodSignature(msg string) { n.Called() }
