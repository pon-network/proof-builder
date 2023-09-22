package builder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/builder/database"
	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
)

func (b *Builder) handleBlockBid(w http.ResponseWriter, req *http.Request) {
	payload := new(builderTypes.BuilderPayloadAttributes)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid payload: %v", err))
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

func (b *Builder) handleBlockBountyBid(w http.ResponseWriter, req *http.Request) {
	payload := new(builderTypes.BuilderPayloadAttributes)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid payload: %v", err))
		return
	}

	resp, err := b.ProcessBuilderBountyBid(payload)
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

	payload := new(commonTypes.VersionedSignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid payload: %v", err))
		return
	}
	
	executionPayload, err := b.SubmitBlindedBlock(*payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	executionPayload_json, err := executionPayload.MarshalJSON()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(executionPayload_json)

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
		Slot           string
		SentAt         string
		MEV            string
		BidAmount      string
		BlockHash      string
		PrivateTxCount string
		TotalTxCount   string
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
	winningBidsBidAmountBigFloat := new(big.Float).SetInt(winningBidsBidAmountBig)
	winningBidsBidAmountBigFloat = new(big.Float).Quo(winningBidsBidAmountBigFloat, big.NewFloat(params.Ether))
	rounded := fmt.Sprintf("%.8f", winningBidsBidAmountBigFloat)
	winningBidsBidAmount, _ = strconv.ParseFloat(rounded, 64)

	var winningBidsMEV float64 = 0
	winningBidsMEVBig, err := b.db.ComputeTotalBlockBidsWonMEV()
	if err != nil {
		log.Error("Error getting total bids won MEV", "error", err)
	}
	winningBidsMEVBigFloat, ok := new(big.Float).SetString(winningBidsMEVBig.String())
	if !ok {
		log.Error("Error converting big.Int to big.Float", "error", err)
	}
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

	avgBidMonthlyBigFloat, err := b.db.ComputeAverageBidAmountGroupByMonth(month5Ago, now, chartLabels)
	var avgBidMonthly []float64 = make([]float64, len(avgBidMonthlyBigFloat))
	if err != nil {
		log.Error("Error getting average bid amount", "error", err)
	}
	for i, v := range avgBidMonthlyBigFloat {
		avgBid := v
		avgBid = new(big.Float).Quo(avgBid, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", avgBid)
		avgBidMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	avgMEVMonthlyBigFloat, err := b.db.ComputeAverageMEVGroupByMonth(month5Ago, now, chartLabels)
	var avgMEVMonthly []float64 = make([]float64, len(avgMEVMonthlyBigFloat))
	if err != nil {
		log.Error("Error getting average MEV", "error", err)
	}
	for i, v := range avgMEVMonthlyBigFloat {
		avgMEV := v
		avgMEV = new(big.Float).Quo(avgMEV, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", avgMEV)
		avgMEVMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	bidsWonSumBidAmountMonthly, err := b.db.ComputeTotalBlockBidsWonBidAmountGroupByMonth(month5Ago, now, chartLabels)
	var bidsWonTotalBidAmountMonthly []float64 = make([]float64, len(bidsWonSumBidAmountMonthly))
	if err != nil {
		log.Error("Error getting bids won total bid amount monthly", "error", err)
	}
	for i, v := range bidsWonSumBidAmountMonthly {
		totalBidAmount := new(big.Float).SetInt(v)
		totalBidAmount = new(big.Float).Quo(totalBidAmount, big.NewFloat(params.Ether))
		rounded := fmt.Sprintf("%.8f", totalBidAmount)
		bidsWonTotalBidAmountMonthly[i], _ = strconv.ParseFloat(rounded, 64)
	}

	bidsWonSumMEVMonthly, err := b.db.ComputeTotalBlockBidsWonMEVGroupByMonth(month5Ago, now, chartLabels)
	var bidsWonTotalMEVMonthly []float64 = make([]float64, len(bidsWonSumMEVMonthly))
	if err != nil {
		log.Error("Error getting bids won total MEV monthly", "error", err)
	}
	for i, v := range bidsWonSumMEVMonthly {
		totalMEV := new(big.Float).SetInt(v)
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
		mevVal := new(big.Float).SetInt(&bid.MEV)
		mevVal.Quo(mevVal, big.NewFloat(params.Ether))

		bidVal := new(big.Float).SetInt(&bid.Value)
		bidVal.Quo(bidVal, big.NewFloat(params.Ether))

		allBidsData = append(allBidsData, bidData{
			bid.Slot.String(),
			bid.InsertedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", mevVal),
			fmt.Sprintf("%.8f", bidVal),
			bid.BlockHash,
			strconv.FormatUint(bid.PriorityTransactionsCount, 10),
			strconv.FormatUint(bid.TransactionsCount, 10),
		})
	}

	var recentBidsWonData []bidData
	for _, bid := range recentBidsWon {
		// MEV and Value need to be converted from wei to eth
		mevVal := new(big.Float).SetInt(&bid.MEV)
		mevVal.Quo(mevVal, big.NewFloat(params.Ether))

		bidVal := new(big.Float).SetInt(&bid.Value)
		bidVal.Quo(bidVal, big.NewFloat(params.Ether))

		recentBidsWonData = append(recentBidsWonData, bidData{
			bid.Slot.String(),
			bid.InsertedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", mevVal),
			fmt.Sprintf("%.8f", bidVal),
			bid.BlockHash,
			strconv.FormatUint(bid.PriorityTransactionsCount, 10),
			strconv.FormatUint(bid.TransactionsCount, 10),
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
