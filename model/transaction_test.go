// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package model

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestTransactionTransform(t *testing.T) {
	id := "123"
	result := "tx result"
	dropped, startedSpans := 5, 14
	name := "mytransaction"

	tests := []struct {
		Transaction Transaction
		Output      common.MapStr
		Msg         string
	}{
		{
			Transaction: Transaction{},
			Output: common.MapStr{
				"id":       "",
				"type":     "",
				"duration": common.MapStr{"us": 0},
				"sampled":  false,
			},
			Msg: "Empty Event",
		},
		{
			Transaction: Transaction{
				ID:       id,
				Type:     "tx",
				Duration: 65.98,
			},
			Output: common.MapStr{
				"id":       id,
				"type":     "tx",
				"duration": common.MapStr{"us": 65980},
				"sampled":  false,
			},
			Msg: "SpanCount empty",
		},
		{
			Transaction: Transaction{
				ID:        id,
				Type:      "tx",
				Duration:  65.98,
				SpanCount: SpanCount{Started: &startedSpans},
			},
			Output: common.MapStr{
				"id":         id,
				"type":       "tx",
				"duration":   common.MapStr{"us": 65980},
				"span_count": common.MapStr{"started": 14},
				"sampled":    false,
			},
			Msg: "SpanCount only contains `started`",
		},
		{
			Transaction: Transaction{
				ID:        id,
				Type:      "tx",
				Duration:  65.98,
				SpanCount: SpanCount{Dropped: &dropped},
			},
			Output: common.MapStr{
				"id":         id,
				"type":       "tx",
				"duration":   common.MapStr{"us": 65980},
				"span_count": common.MapStr{"dropped": 5},
				"sampled":    false,
			},
			Msg: "SpanCount only contains `dropped`",
		},
		{
			Transaction: Transaction{
				ID:        id,
				Name:      name,
				Type:      "tx",
				Result:    result,
				Timestamp: time.Now(),
				Duration:  65.98,
				Sampled:   true,
				SpanCount: SpanCount{Started: &startedSpans, Dropped: &dropped},
			},
			Output: common.MapStr{
				"id":         id,
				"name":       "mytransaction",
				"type":       "tx",
				"result":     "tx result",
				"duration":   common.MapStr{"us": 65980},
				"span_count": common.MapStr{"started": 14, "dropped": 5},
				"sampled":    true,
			},
			Msg: "Full Event",
		},
	}

	for idx, test := range tests {
		output := test.Transaction.toBeatEvent()
		assert.Equal(t, test.Output, output.Fields["transaction"], fmt.Sprintf("Failed at idx %v; %s", idx, test.Msg))
	}
}

func TestTransactionTransformOutcome(t *testing.T) {
	tx := Transaction{Outcome: "success"}
	event := tx.toBeatEvent()
	assert.Equal(t, common.MapStr{"outcome": "success"}, event.Fields["event"])
}

func TestEventsTransformWithMetadata(t *testing.T) {
	hostname := "a.b.c"
	architecture := "darwin"
	platform := "x64"
	timestamp := time.Date(2019, 1, 3, 15, 17, 4, 908.596*1e6, time.FixedZone("+0100", 3600))
	timestampUs := timestamp.UnixNano() / 1000
	id, name, ip, userAgent := "123", "jane", "63.23.123.4", "node-js-2.3"
	url, referer := "https://localhost", "http://localhost"
	serviceName, serviceNodeName, serviceVersion := "myservice", "service-123", "2.1.3"
	eventMetadata := Metadata{
		Service: Service{
			Name:    serviceName,
			Version: serviceVersion,
			Node:    ServiceNode{Name: serviceNodeName},
		},
		System: System{
			ConfiguredHostname: name,
			DetectedHostname:   hostname,
			Architecture:       architecture,
			Platform:           platform,
		},
		User:      User{ID: id, Name: name},
		UserAgent: UserAgent{Original: userAgent},
		Client:    Client{IP: net.ParseIP(ip)},
		Labels:    common.MapStr{"a": true},
	}

	request := HTTPRequest{Method: "post", Headers: common.MapStr{}, Referrer: referer}
	response := HTTPResponse{Finished: new(bool), Headers: common.MapStr{"content-type": []string{"text/html"}}}
	txWithContext := Transaction{
		Metadata:  eventMetadata,
		Timestamp: timestamp,
		Labels:    common.MapStr{"a": "b"},
		Page:      &Page{URL: &URL{Original: url}, Referer: referer},
		HTTP:      &HTTP{Request: &request, Response: &response},
		URL:       &URL{Original: url},
		Custom:    common.MapStr{"foo.bar": "baz"},
		Message:   &Message{QueueName: "routeUser"},
		Sampled:   true,
	}
	event := txWithContext.toBeatEvent()
	assert.Equal(t, common.MapStr{
		"user":       common.MapStr{"id": "123", "name": "jane"},
		"client":     common.MapStr{"ip": ip},
		"source":     common.MapStr{"ip": ip},
		"user_agent": common.MapStr{"original": userAgent},
		"host": common.MapStr{
			"architecture": "darwin",
			"hostname":     "a.b.c",
			"name":         "jane",
			"os": common.MapStr{
				"platform": "x64",
			},
		},
		"processor": common.MapStr{
			"event": "transaction",
			"name":  "transaction",
		},
		"service": common.MapStr{
			"name":    serviceName,
			"version": serviceVersion,
			"node":    common.MapStr{"name": serviceNodeName},
		},
		"timestamp": common.MapStr{"us": timestampUs},
		"transaction": common.MapStr{
			"duration": common.MapStr{"us": 0},
			"id":       "",
			"type":     "",
			"sampled":  true,
			"page":     common.MapStr{"url": url, "referer": referer},
			"custom": common.MapStr{
				"foo_bar": "baz",
			},
			"message": common.MapStr{"queue": common.MapStr{"name": "routeUser"}},
		},
		"event":  common.MapStr{"outcome": ""},
		"labels": common.MapStr{"a": "b"},
		"url":    common.MapStr{"original": url},
		"http": common.MapStr{
			"request":  common.MapStr{"method": "post", "referrer": referer},
			"response": common.MapStr{"finished": false, "headers": common.MapStr{"content-type": []string{"text/html"}}},
		},
	}, event.Fields)
}

func TestTransformTransactionHTTP(t *testing.T) {
	request := HTTPRequest{Method: "post", Body: "<html><marquee>hello world</marquee></html>"}
	tx := Transaction{
		HTTP: &HTTP{Request: &request},
	}
	event := tx.toBeatEvent()
	assert.Equal(t, common.MapStr{
		"request": common.MapStr{
			"method":        request.Method,
			"body.original": request.Body,
		},
	}, event.Fields["http"])
}

func TestTransactionTransformPage(t *testing.T) {
	id := "123"
	urlExample := "http://example.com/path"

	tests := []struct {
		Transaction Transaction
		Output      common.MapStr
		Msg         string
	}{
		{
			Transaction: Transaction{
				ID:        id,
				Type:      "tx",
				Timestamp: time.Now(),
				Duration:  65.98,
				URL:       ParseURL("https://localhost:8200/", "", ""),
				Page: &Page{
					URL: ParseURL(urlExample, "", ""),
				},
			},
			Output: common.MapStr{
				"domain":   "localhost",
				"full":     "https://localhost:8200/",
				"original": "https://localhost:8200/",
				"path":     "/",
				"port":     8200,
				"scheme":   "https",
			},
			Msg: "With Page URL and Request URL",
		},
	}

	for idx, test := range tests {
		output := test.Transaction.toBeatEvent()
		assert.Equal(t, test.Output, output.Fields["url"], fmt.Sprintf("Failed at idx %v; %s", idx, test.Msg))
	}
}

func TestTransactionTransformMarks(t *testing.T) {
	tests := []struct {
		Transaction Transaction
		Output      common.MapStr
		Msg         string
	}{
		{
			Transaction: Transaction{
				Marks: TransactionMarks{
					"a.b": TransactionMark{
						"c.d": 123,
					},
				},
			},
			Output: common.MapStr{
				"a_b": common.MapStr{
					"c_d": common.Float(123),
				},
			},
			Msg: "Unsanitized transaction mark names",
		},
	}

	for idx, test := range tests {
		output := test.Transaction.toBeatEvent()
		marks, _ := output.Fields.GetValue("transaction.marks")
		assert.Equal(t, test.Output, marks, fmt.Sprintf("Failed at idx %v; %s", idx, test.Msg))
	}
}

func TestTransactionSession(t *testing.T) {
	tests := []struct {
		Transaction Transaction
		Output      common.MapStr
	}{{
		Transaction: Transaction{
			Session: TransactionSession{
				ID: "session_id",
			},
		},
		Output: common.MapStr{
			"id": "session_id",
		},
	}, {
		Transaction: Transaction{
			Session: TransactionSession{
				ID:       "session_id",
				Sequence: 123,
			},
		},
		Output: common.MapStr{
			"id":       "session_id",
			"sequence": 123,
		},
	}, {
		Transaction: Transaction{
			Session: TransactionSession{
				// Sequence is ignored if ID is empty.
				Sequence: 123,
			},
		},
		Output: nil,
	}}

	for _, test := range tests {
		output := test.Transaction.toBeatEvent()
		session, err := output.Fields.GetValue("session")
		if test.Output == nil {
			assert.Equal(t, common.ErrKeyNotFound, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.Output, session)
		}
	}
}
