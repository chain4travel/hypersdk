// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	ametrics "github.com/ava-labs/avalanchego/api/metrics"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/chain4travel/hypersdk/examples/touristicvm/consts"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	transfer    prometheus.Counter
	createAsset prometheus.Counter
	mintAsset   prometheus.Counter

	createNFT   prometheus.Counter
	transferNFT prometheus.Counter
	importAsset prometheus.Counter
	exportAsset prometheus.Counter
}

func newMetrics(gatherer ametrics.MultiGatherer) (*metrics, error) {
	m := &metrics{
		transfer: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "transfer",
			Help:      "number of transfer actions",
		}),
		createAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "create_asset",
			Help:      "number of create asset actions",
		}),
		mintAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "mint_asset",
			Help:      "number of mint asset actions",
		}),
		createNFT: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "create_nft",
			Help:      "number of create nft actions",
		}),
		transferNFT: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "transfer_nft",
			Help:      "number of create transfer nft actions",
		}),
		importAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "import_asset",
			Help:      "number of import asset actions",
		}),
		exportAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "export_asset",
			Help:      "number of export asset actions",
		}),
	}
	r := prometheus.NewRegistry()
	errs := wrappers.Errs{}
	errs.Add(
		r.Register(m.transfer),
		r.Register(m.createAsset),
		r.Register(m.mintAsset),
		r.Register(m.createNFT),
		r.Register(m.transferNFT),
		r.Register(m.importAsset),
		r.Register(m.exportAsset),
		gatherer.Register(consts.Name, r),
	)
	return m, errs.Err
}
