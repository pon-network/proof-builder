package builder

func (b *Builder) DataCleanUp() {
	b.slotMu.Lock()
	defer b.slotMu.Unlock()
	b.slotSubmissionsLock.Lock()
	defer b.slotSubmissionsLock.Unlock()

	b.beacon.BeaconData.Mu.Lock()
	currentSlot := b.beacon.BeaconData.CurrentSlot
	b.beacon.BeaconData.Mu.Unlock()

	// Check if execution payload cache has any old slots since we are in a new slot bid
	for slot := range b.executionPayloadCache {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.executionPayloadCache, slot)
		}
	}

	for slot := range b.slotSubmissions {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotSubmissions, slot)
		}
	}

	for slot := range b.slotSubmissionsChan {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotSubmissionsChan, slot)
			delete(b.slotBidCompleteChan, slot)
			delete(b.slotBountyCompleteChan, slot)
		}
	}

	for slot := range b.slotAttrs {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotAttrs, slot)
		}
	}

	for slot := range b.slotBountyAttrs {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBountyAttrs, slot)
		}
	}

	for slot := range b.slotBidAmounts {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBidAmounts, slot)
		}
	}

	for slot := range b.slotBountyAmount {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBountyAmount, slot)
		}
	}

}