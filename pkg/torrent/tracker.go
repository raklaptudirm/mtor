// Copyright © 2021 Rak Laptudirm <raklaptudirm@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package torrent

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/raklaptudirm/mtor/pkg/peer"
)

// trackerResponse represents a response from the tracker.
type trackerResponse struct {
	Failure string `bencode:"failure reason"`  // failure message
	Warning string `bencode:"warning message"` // warning message

	Interval   int `bencode:"interval"`     // interval to reconnect after
	MinIntrval int `bencode:"min interval"` // minimum interval to reconnect after

	TrackerID string `bencode:"tracker id"` // id of the tracker

	CompletePeers   int `bencode:"complete"`   // number of peers with complete pieces
	IncompletePeers int `bencode:"incomplete"` // number of peers with incomplete pieces

	Peers string `bencode:"peers"` // compact peer ips and ports
}

// requestTracker requests to t's tracker and returns the parsed response.
func (t *Torrent) requestTracker() (*trackerResponse, error) {
	url, err := t.Tracker()
	if err != nil {
		return nil, err
	}

	// tracker connection client
	c := &http.Client{Timeout: 5 * time.Second}

	// get peerlist from tracker
	res, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var trackerRes trackerResponse
	// unmarshal bencode response
	err = bencode.Unmarshal(res.Body, &trackerRes)
	if err != nil {
		return nil, err
	}

	return &trackerRes, nil
}

// Peers returns a list of peers to fetch pieces from.
func (t *Torrent) Peers() ([]peer.Peer, error) {
	// get response from tracker
	res, err := t.requestTracker()
	if err != nil {
		return nil, err
	}

	if res.Failure != "" {
		return nil, errors.New(res.Failure)
	}

	peerBuf := []byte(res.Peers)
	// unmarshal compact peerlist
	return peer.Unmarshal(peerBuf)
}

// Tracker returns the url of t's tracker, along with parameters.
func (t *Torrent) Tracker() (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", err
	}

	// set url params
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},     // infohash of torrent
		"peer_id":    []string{string(t.Name[:])},         // client's peer id
		"port":       []string{strconv.Itoa(int(t.Port))}, // port client is listening on
		"uploaded":   []string{"0"},                       // number of bytes uploaded
		"downloaded": []string{"0"},                       // number of bytes downloaded
		"left":       []string{strconv.Itoa(t.Length)},    // number of bytes left to download
		"compact":    []string{"1"},                       // 1 to get peerlist be in compact format
		"numwant":    []string{"500"},                     // request 500 peers
	}
	base.RawQuery = params.Encode()

	return base.String(), nil
}
