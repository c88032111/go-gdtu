// Copyright 2015 The go-gdtu Authors
// This file is part of the go-gdtu library.
//
// The go-gdtu library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gdtu library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// algdtu with the go-gdtu library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/c88032111/go-gdtu/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("gdtu/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("gdtu/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("gdtu/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("gdtu/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("gdtu/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("gdtu/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("gdtu/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("gdtu/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("gdtu/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("gdtu/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("gdtu/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("gdtu/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("gdtu/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("gdtu/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("gdtu/downloader/throttle", nil)
)
