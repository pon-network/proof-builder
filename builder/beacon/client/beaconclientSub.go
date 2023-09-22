package beaconClient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/r3labs/sse/v2"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
)

func (b *beaconClient) SubscribeToHeadEvents(ctx context.Context, headChannel chan beaconTypes.HeadEventData) {
	/*
		Subscribe to head events from the beacon chain
		Events are sent to the headChannel
	*/
	log.Info("starting head events subscription", "endpoint", b.beaconEndpoint.String())
	defer log.Debug("head events subscription ended", "endpoint", b.beaconEndpoint.String())

	for {
		client := sse.NewClient(fmt.Sprintf("%s/eth/v1/events?topics=head", b.beaconEndpoint.String()))
		// Use sse client to subscribe to events
		err := client.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			var event beaconTypes.HeadEventData
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Warn("head event subscription failed", "error", err)
				return
			}
			headChannel <- event
		})

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		if err != nil {
			log.Error("failed to subscribe to head events")
			time.Sleep(1 * time.Second)
		}

		log.Warn("beaconclient SubscribeRaw ended, reconnecting")
	}

}

func (b *beaconClient) SubscribeToPayloadAttributesEvents(ctx context.Context, payloadAttributesC chan beaconTypes.PayloadAttributesEvent) {
	/*
		Subscribe to payload attributes events from the beacon chain
		Events are sent to the payloadAttributesC channel
	*/
	log.Info("starting payload attributes events subscription", "endpoint", b.beaconEndpoint.String())
	defer log.Debug("payload attributes events subscription ended", "endpoint", b.beaconEndpoint.String())

	for {
		client := sse.NewClient(fmt.Sprintf("%s/eth/v1/events?topics=payload_attributes", b.beaconEndpoint.String()))
		// Use sse client to subscribe to events
		err := client.SubscribeRawWithContext(ctx,func(msg *sse.Event) {
			var event beaconTypes.PayloadAttributesEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Warn("payload event subscription failed", "error", err)
				return
			}

			if event.Data == nil {
				log.Warn("payload event subscription failed", "error", "payload data is nil")
				return
			}
			payloadAttributesC <- event

		})
		if err != nil {
			log.Error("failed to subscribe to payload_attributes events")
			time.Sleep(1 * time.Second)
		}
		log.Warn("beaconclient SubscribeRaw ended, reconnecting")
	}
}
