package buyer

import (
	"log"
	"time"

	"github.com/mtlynch/gofn-prosper/prosper"
	"github.com/mtlynch/gofn-prosper/types"
)

// TODO: Add support in Polling for excluding based on a blacklist of
// employment status. Do I actually need to do this? Maybe I can just
// whitelist employment statuses.

func Poll(checkInterval time.Duration, f prosper.SearchFilter, isBuyingEnabled bool, c *prosper.Client) error {
	allListings := make(chan types.Listing)
	newListings := make(chan types.Listing)
	orders := make(chan types.OrderID)
	orderUpdates := make(chan types.OrderResponse)
	listingPoller := listingPoller{
		s:            c,
		searchFilter: f,
		listings:     allListings,
		pollInterval: checkInterval,
		clock:        types.DefaultClock{},
	}
	seenFilter, err := NewSeenListingFilter(allListings, newListings)
	if err != nil {
		return err
	}

	var buyer listingBuyer
	var tracker orderTracker
	var logger orderStatusLogger
	if isBuyingEnabled {
		buyer = listingBuyer{
			listings:  newListings,
			orders:    orders,
			bidPlacer: c,
			bidAmount: 25.0,
		}
		tracker = orderTracker{
			querier:      c,
			orders:       orders,
			orderUpdates: orderUpdates,
		}
		logger, err = NewOrderStatusLogger(orderUpdates)
		if err != nil {
			log.Printf("failed to create order status logger: %v", err)
			return err
		}
	}
	go func() {
		log.Printf("starting buyer polling")

		go listingPoller.Run()
		go seenFilter.Run()
		if isBuyingEnabled {
			go buyer.Run()
			go tracker.Run()
			go logger.Run()
		} else {
			l := <-newListings
			log.Printf("new purchase candidate: %v", l.ListingNumber)
		}
	}()

	return nil
}
