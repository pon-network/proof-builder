package builder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	capellaApi "github.com/attestantio/go-eth2-client/api/v1/capella"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/builder/database"
	builderTypes "github.com/ethereum/go-ethereum/builder/types"
)

func (b *Builder) handlePrivateTransactions(w http.ResponseWriter, req *http.Request) {
	payload := new(builderTypes.PrivateTransactionsPayload)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		fmt.Println("error decoding payload", err)
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	resp, err := b.OnPrivateTransactions(payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Builder) handleBlockBid(w http.ResponseWriter, req *http.Request) {
	payload := new(builderTypes.BuilderPayloadAttributes)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		fmt.Println("error decoding payload", err)
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	resp, err := b.ProcessBuilderBid(payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Builder) handleBlindedBlockSubmission(w http.ResponseWriter, req *http.Request) {

	payload := new(capellaApi.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		fmt.Println("error decoding payload", err)
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if payload.Message == nil {
		respondError(w, http.StatusBadRequest, "invalid payload, message is nil")
		return
	}

	log.Info("Blinded block submission request", "payloadMessage", *&payload.Message.Slot, "payloadSignature", payload.Signature)

	executionPayload, err := b.SubmitBlindedBlock(*payload.Message, payload.Signature)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(executionPayload); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Builder) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode("PoN Builder v1"); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Builder) handleIndex(w http.ResponseWriter, req *http.Request) {

	type bidData struct {
		Slot           uint64
		SentAt         string
		MEV            string
		BidAmount      string
		BlockHash      string
		PrivateTxCount uint64
		TotalTxCount   uint64
	}

	indexTemplate, err := parseIndexTemplate()
	if err != nil {
		http.Error(w, "not available", http.StatusInternalServerError)
	}

	syncStatus := "Syncing"
	if b.eth.Synced() {
		syncStatus = "Synced"
	}

	relayUrl := b.relay.GetEndpoint()

	relayStatus := "UNAVAILABLE"
	// Check if b.relay.CheckStatus() doesnt return an error
	if b.relay.CheckStatus() == nil {
		relayStatus = "OK"
	}

	metricsEnabled := "False"
	if b.MetricsEnabled {
		metricsEnabled = "True"
	}

	var totalBidsSent uint64 = 0
	totalBidsSent, err = b.db.CountTotalBlockBids()
	if err != nil {
		log.Error("Error getting total bids sent", "error", err)
	}

	var totalBidsWon uint64 = 0
	totalBidsWon, err = b.db.CountTotalBlockBidsWon()
	if err != nil {
		log.Error("Error getting total bids won", "error", err)
	}

	var winningBidsBidAmount float64 = 0
	winningBidsBidAmountBig, err := b.db.ComputeTotalBlockBidsWonBidAmount()
	if err != nil {
		log.Error("Error getting total bids won value", "error", err)
	}
	winningBidsBidAmountBigFloat := new(big.Float).SetFloat64(winningBidsBidAmountBig)
	winningBidsBidAmountBigFloat = new(big.Float).Quo(winningBidsBidAmountBigFloat, big.NewFloat(params.Ether))
	rounded := fmt.Sprintf("%.8f", winningBidsBidAmountBigFloat)
	winningBidsBidAmount, _ = strconv.ParseFloat(rounded, 64)

	var winningBidsMEV float64 = 0
	winningBidsMEVBig, err := b.db.ComputeTotalBlockBidsWonMEV()
	if err != nil {
		log.Error("Error getting total bids won MEV", "error", err)
	}
	winningBidsMEVBigFloat := new(big.Float).SetFloat64(winningBidsMEVBig)
	winningBidsMEVBigFloat = new(big.Float).Quo(winningBidsMEVBigFloat, big.NewFloat(params.Ether))
	rounded = fmt.Sprintf("%.8f", winningBidsMEVBigFloat)
	winningBidsMEV, _ = strconv.ParseFloat(rounded, 64)

	var totalBidsSubmittedToChain uint64 = 0
	totalBidsSubmittedToChain, err = b.db.CountTotalBlockBidsSubmittedToChain()
	if err != nil {
		log.Error("Error getting total bids submitted to chain", "error", err)
	}

	var allBids []database.BuilderBlockBidEntry
	allBids, err = b.db.GetBlockBidsPaginated(0, 1000)
	if err != nil {
		log.Error("Error getting bids", "error", err)
	}

	var recentBidsWon []database.BuilderBlockBidEntry
	recentBidsWon, err = b.db.GetBlockBidsWonPaginated(0, 20)
	if err != nil {
		log.Error("Error getting recent bids won", "error", err)
	}

	now := time.Now()
	nowDate := time.Date(now.Year(), now.Month(), 15,
		now.Hour(), now.Minute(), now.Second(), now.Nanosecond(),
		now.Location())
	month5Ago := nowDate.AddDate(0, -4, 0)

	var chartLabels []string
	for i := 0; i < 5; i++ {
		chartLabels = append(chartLabels, month5Ago.AddDate(0, i, 0).Format("Jan"))
	}

	month5Ago = time.Date(month5Ago.Year(), month5Ago.Month(), 1,
		month5Ago.Hour(), month5Ago.Minute(), month5Ago.Second(), month5Ago.Nanosecond(),
		month5Ago.Location())

	var avgBidMonthly []float64
	avgBidMonthly, err = b.db.ComputeAverageBidAmountGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting average bid amount", "error", err)
	}
	for i, v := range avgBidMonthly {
		avgBid := new(big.Float).SetFloat64(v)
		avgBid = new(big.Float).Quo(avgBid, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", avgBid)
		avgBidMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	var avgMEVMonthly []float64
	avgMEVMonthly, err = b.db.ComputeAverageMEVGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting average MEV", "error", err)
	}
	for i, v := range avgMEVMonthly {
		avgMEV := new(big.Float).SetFloat64(v)
		avgMEV = new(big.Float).Quo(avgMEV, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", avgMEV)
		avgMEVMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	var bidsWonTotalBidAmountMonthly []float64
	bidsWonTotalBidAmountMonthly, err = b.db.ComputeTotalBlockBidsWonBidAmountGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting bids won total bid amount monthly", "error", err)
	}
	for i, v := range bidsWonTotalBidAmountMonthly {
		totalBidAmount := new(big.Float).SetFloat64(v)
		totalBidAmount = new(big.Float).Quo(totalBidAmount, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", totalBidAmount)
		bidsWonTotalBidAmountMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	var bidsWonTotalMEVMonthly []float64
	bidsWonTotalMEVMonthly, err = b.db.ComputeTotalBlockBidsWonMEVGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting bids won total MEV monthly", "error", err)
	}
	for i, v := range bidsWonTotalMEVMonthly {
		totalMEV := new(big.Float).SetFloat64(v)
		totalMEV = new(big.Float).Quo(totalMEV, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", totalMEV)
		bidsWonTotalMEVMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	var bidsSentMonthly []uint64
	bidsSentMonthly, err = b.db.CountTotalBidsGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting bids sent monthly", "error", err)
	}

	var bidsWonMonthly []uint64
	bidsWonMonthly, err = b.db.CountTotalBidsWonGroupByMonth(month5Ago, now, chartLabels)
	if err != nil {
		log.Error("Error getting bids won monthly", "error", err)
	}

	var allBidsData []bidData
	for _, bid := range allBids {
		// MEV and Value need to be converted from wei to eth
		mevVal := new(big.Float).SetUint64(bid.MEV)
		mevVal.Quo(mevVal, big.NewFloat(params.Ether))

		bidVal := new(big.Float).SetUint64(bid.Value)
		bidVal.Quo(bidVal, big.NewFloat(params.Ether))

		allBidsData = append(allBidsData, bidData{
			bid.Slot,
			bid.InsertedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", mevVal),
			fmt.Sprintf("%.8f", bidVal),
			bid.BlockHash,
			bid.PriorityTransactionsCount,
			bid.TransactionsCount,
		})
	}

	var recentBidsWonData []bidData
	for _, bid := range recentBidsWon {
		// MEV and Value need to be converted from wei to eth
		mevVal := new(big.Float).SetUint64(bid.MEV)
		mevVal.Quo(mevVal, big.NewFloat(params.Ether))

		bidVal := new(big.Float).SetUint64(bid.Value)
		bidVal.Quo(bidVal, big.NewFloat(params.Ether))

		recentBidsWonData = append(recentBidsWonData, bidData{
			bid.Slot,
			bid.InsertedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", mevVal),
			fmt.Sprintf("%.8f", bidVal),
			bid.BlockHash,
			bid.PriorityTransactionsCount,
			bid.TransactionsCount,
		})
	}

	// Recent bids are the first 20 bids
	var recentBidsData []bidData

	if len(allBidsData) > 20 {
		recentBidsData = allBidsData[:20]
	} else {
		recentBidsData = allBidsData
	}

	statusData := struct {
		SyncStatus                  string
		RelayUrl                    string
		RelayStatus                 string
		MetricsEnabled              string
		TotalBidsSent               uint64
		TotalBidsWon                uint64
		TotalBidsSubmittedToChain   uint64
		WinningBidsBidAmount        float64
		WinningBidsMEV              float64
		RecentBids                  []bidData
		RecentBidsWon               []bidData
		AllBids                     []bidData
		ChartAvgBidAmounts          []float64
		ChartAvgMEVs                []float64
		ChartBidsSent               []uint64
		ChartTotalBidsWonBidAmounts []float64
		ChartTotalBidsWonMEVs       []float64
		ChartBidsWon                []uint64
		ChartLabels                 []string
	}{
		syncStatus,
		relayUrl,
		relayStatus,
		metricsEnabled,
		totalBidsSent,
		totalBidsWon,
		totalBidsSubmittedToChain,
		winningBidsBidAmount,
		winningBidsMEV,
		recentBidsData,
		recentBidsWonData,
		allBidsData,
		avgBidMonthly,
		avgMEVMonthly,
		bidsSentMonthly,
		bidsWonTotalBidAmountMonthly,
		bidsWonTotalMEVMonthly,
		bidsWonMonthly,
		chartLabels,
	}

	if err := indexTemplate.Execute(w, statusData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
