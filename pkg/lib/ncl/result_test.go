//go:build unit || !integration

package ncl

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

type ResultTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
}

func (suite *ResultTestSuite) SetupSuite() {
	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *ResultTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *ResultTestSuite) TestResultCreation() {
	testCases := []struct {
		name          string
		setupFn       func() *Result
		expectedError string
		expectedDelay time.Duration
		verifyFn      func(*Result)
	}{
		{
			name: "new empty result",
			setupFn: func() *Result {
				return NewResult()
			},
		},
		{
			name: "result with error",
			setupFn: func() *Result {
				return NewResult().WithError(errors.New("test error"))
			},
			expectedError: "test error",
		},
		{
			name: "result with delay",
			setupFn: func() *Result {
				return NewResult().WithDelay(5 * time.Second)
			},
			expectedDelay: 5 * time.Second,
		},
		{
			name: "result with error and delay",
			setupFn: func() *Result {
				return NewResult().
					WithError(errors.New("test error")).
					WithDelay(5 * time.Second)
			},
			expectedError: "test error",
			expectedDelay: 5 * time.Second,
		},
		{
			name: "result with nil error",
			setupFn: func() *Result {
				return NewResult().WithError(nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := tc.setupFn()

			// Verify error
			if tc.expectedError != "" {
				suite.Equal(tc.expectedError, result.Error)
				suite.Error(result.Err())
				suite.Equal(tc.expectedError, result.Err().Error())
			} else {
				suite.Empty(result.Error)
				suite.NoError(result.Err())
			}

			// Verify delay
			suite.Equal(tc.expectedDelay, result.DelayDuration())

			if tc.verifyFn != nil {
				tc.verifyFn(result)
			}
		})
	}
}

func (suite *ResultTestSuite) TestResultSerialization() {
	testCases := []struct {
		name   string
		result *Result
	}{
		{
			name:   "empty result",
			result: NewResult(),
		},
		{
			name:   "result with error",
			result: NewResult().WithError(errors.New("test error")),
		},
		{
			name:   "result with delay",
			result: NewResult().WithDelay(5 * time.Second),
		},
		{
			name: "result with error and delay",
			result: NewResult().
				WithError(errors.New("test error")).
				WithDelay(5 * time.Second),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Marshal
			data, err := json.Marshal(tc.result)
			suite.Require().NoError(err)

			// Unmarshal
			var decoded Result
			err = json.Unmarshal(data, &decoded)
			suite.Require().NoError(err)

			// Verify fields match
			suite.Equal(tc.result.Error, decoded.Error)
			suite.Equal(tc.result.Delay, decoded.Delay)
		})
	}
}

func (suite *ResultTestSuite) TestAckNack() {
	testCases := []struct {
		name        string
		action      func(*nats.Msg) error
		verifyReply func(*Result)
	}{
		{
			name:   "ack with reply",
			action: Ack,
			verifyReply: func(r *Result) {
				suite.Empty(r.Error)
				suite.Zero(r.Delay)
			},
		},
		{
			name: "nack with error",
			action: func(m *nats.Msg) error {
				return Nack(m, errors.New("test error"))
			},
			verifyReply: func(r *Result) {
				suite.Equal("test error", r.Error)
				suite.Zero(r.Delay)
			},
		},
		{
			name: "nack with error and delay",
			action: func(m *nats.Msg) error {
				return NackWithDelay(m, errors.New("test error"), 5*time.Second)
			},
			verifyReply: func(r *Result) {
				suite.Equal("test error", r.Error)
				suite.Equal(5*time.Second, r.DelayDuration())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create subscriber that will send responses using the test action
			subj := "test.subject"
			sub, err := suite.natsConn.Subscribe(subj, func(msg *nats.Msg) {
				err := tc.action(msg)
				suite.Require().NoError(err)
			})
			suite.Require().NoError(err)
			defer sub.Unsubscribe()

			// Publish message and wait for response
			reply := suite.natsConn.NewRespInbox()
			replySub, err := suite.natsConn.SubscribeSync(reply)
			suite.Require().NoError(err)
			defer replySub.Unsubscribe()

			err = suite.natsConn.PublishRequest(subj, reply, []byte("test"))
			suite.Require().NoError(err)

			// Verify response
			replyMsg, err := replySub.NextMsg(100 * time.Millisecond)
			suite.Require().NoError(err, "should receive reply")

			result, err := ParseResult(replyMsg)
			suite.Require().NoError(err)
			tc.verifyReply(result)
		})
	}
}

func (suite *ResultTestSuite) TestNoReplySubject() {
	// Create subscriber that will try to Ack
	subj := "test.subject"
	sub, err := suite.natsConn.Subscribe(subj, func(msg *nats.Msg) {
		err := Ack(msg)
		suite.NoError(err)
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	// Publish without reply subject
	err = suite.natsConn.Publish(subj, []byte("test"))
	suite.NoError(err)

	// Give time for message to be processed
	time.Sleep(50 * time.Millisecond)
}

func (suite *ResultTestSuite) TestParseResultErrors() {
	testCases := []struct {
		name        string
		msg         *nats.Msg
		expectedErr error
	}{
		{
			name: "empty data",
			msg:  &nats.Msg{Data: []byte{}},
			expectedErr: fmt.Errorf("empty message data with headers: %v",
				nats.Header{}),
		},
		{
			name: "empty data with no responders",
			msg: &nats.Msg{
				Data: []byte{},
				Header: nats.Header{
					statusHdr: []string{noResponders},
				},
			},
			expectedErr: nats.ErrNoResponders,
		},
		{
			name: "invalid json",
			msg:  &nats.Msg{Data: []byte("invalid")},
			expectedErr: fmt.Errorf("failed to parse result: invalid character 'i' " +
				"looking for beginning of value"),
		},
		{
			name: "wrong type",
			msg:  &nats.Msg{Data: []byte(`{"error": 123}`)},
			expectedErr: fmt.Errorf("failed to parse result: json: cannot unmarshal " +
				"number into Go struct field Result.error of type string"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := ParseResult(tc.msg)
			suite.Error(err)
			if errors.Is(tc.expectedErr, nats.ErrNoResponders) {
				suite.ErrorIs(err, nats.ErrNoResponders)
			} else {
				suite.Equal(tc.expectedErr.Error(), err.Error())
			}
			suite.Nil(result)
		})
	}
}

func TestResultTestSuite(t *testing.T) {
	suite.Run(t, new(ResultTestSuite))
}
