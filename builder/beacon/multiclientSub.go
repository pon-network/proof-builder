package multibeaconClient

import (
	"context"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/log"
)

func (b *MultiBeaconClient) SubscribeToHeadEvents(ctx context.Context, headChannel chan beaconTypes.HeadEventData) {
	/*
		Subscribe to head events using all clients
		No penalty for multiple subscriptions
		Increased reliability in case of some clients being down
	*/
	for _, client := range b.Clients {
		go client.Node.SubscribeToHeadEvents(ctx, headChannel)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case slotHead := <-headChannel:
			log.Info("Received head event", "slot", slotHead.Slot, "blockHash", slotHead.Block)
			// check if head is already processed from another client
			b.BeaconData.Mu.Lock()
			currentSlot := b.BeaconData.CurrentSlot
			b.BeaconData.Mu.Unlock()

			if slotHead.Slot <= currentSlot {
				// head already processed, do not process/contaminate the data
				continue
			}

			b.BeaconData.Mu.Lock()
			b.BeaconData.CurrentHead = slotHead
			b.BeaconData.CurrentSlot = slotHead.Slot
			b.BeaconData.Mu.Unlock()
			go b.SyncStatus()

			// Attempt to get the randao for this slot, the previous slot and the next slot
			for i := uint64(0); i < 3; i++ {
				// current slot -1, current slot, current slot +1
				go b.UpdateRandaoMap(slotHead.Slot - 1 + i)
			}

			// update proposer map
			// check if the current slot is at the edge of an epoch either behind or just infront
			// if so update the proposer map
			currentSlot = slotHead.Slot
			currentEpoch := currentSlot / 32

			if (currentSlot+1)/32 != currentEpoch || (currentSlot-1)/32 != currentEpoch {
				// We are at the edge of an epoch, update the proposer map
				// currentSolot+1 is the first slot of the next epoch means head at the end of the current epoch
				// currentSlot-1 is the last slot of the previous epoch means head at the start of the current epoch
				go b.UpdateValidatorMap()
			}

			// Clean up the proposer map for slots that are older than 2 epochs
			// This is to prevent the map from growing too large
			// We only need to keep the proposer map for the current epoch and the next epoch
			// as we only need to know the proposers for the current epoch and the next epoch
			// to be able to verify the signature of the block
			b.BeaconData.Mu.Lock()
			for k := range b.BeaconData.SlotProposerMap {
				if int64(k) < int64(currentSlot)-64 {
					delete(b.BeaconData.SlotProposerMap, k)
				}
			}
			for k := range b.BeaconData.RandaoMap {
				if int64(k) < int64(currentSlot)-64 {
					delete(b.BeaconData.RandaoMap, k)
				}
			}
			b.BeaconData.Mu.Unlock()

		}
	}

}

func (b *MultiBeaconClient) SubscribeToPayloadAttributesEvents(ctx context.Context, attrsC chan beaconTypes.PayloadAttributesEventData) {
	/*
		Subscribe to payload attributes events using all clients
		No penalty for multiple subscriptions
		Increased reliability in case of some clients being down
	*/
	for _, client := range b.Clients {
		go client.Node.SubscribeToPayloadAttributesEvents(ctx, attrsC)
	}

	for {
		select {
		case payloadAttrs := <-attrsC:

			log.Info("Received payload attributes event", 
			"slot", payloadAttrs.ProposalSlot, 
			"withdrawals", len(payloadAttrs.PayloadAttributes.Withdrawals), 
			"proposer_index", payloadAttrs.ProposerIndex)

			b.BeaconData.Mu.Lock()
			b.BeaconData.SlotPayloadAttributesMap[payloadAttrs.ProposalSlot] = payloadAttrs
			b.BeaconData.Mu.Unlock()

			// Clean up old data
			b.BeaconData.Mu.Lock()
			for slot := range b.BeaconData.SlotPayloadAttributesMap {
				if int64(slot) < int64(payloadAttrs.ProposalSlot)-64 {
					delete(b.BeaconData.SlotPayloadAttributesMap, slot)
				}
			}
			b.BeaconData.Mu.Unlock()

		}
	}

}
