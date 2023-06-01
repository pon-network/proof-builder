package builder

import (
	"html/template"
)

func parseIndexTemplate() (*template.Template, error) {
	return template.New("index").Parse(
		`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Blockswap PoN Builder</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <style>

    body {
      background-color: black;
      font-family: Arial, sans-serif;
      color: white;
    }

    .heading {
        font-size: 2.25rem;
        line-height: 2.5rem;
        font-weight: 600;
        color: white;
        display: flex;
        padding-top: 40px;
        align-items: center;
        justify-content: center;
    }
    
    .container {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 20px;
      margin: 20px;
    }

    .containerFull {
        margin: 20px;
    }

    .box {
      border-radius: 10px;
      padding: 20px;
      background-color: #2d2e35;
      box-shadow: 5px 5px 10px rgba(0, 0, 0, 0.5);
    }

    .tableFixed::-webkit-scrollbar-track
    {
      -webkit-box-shadow: inset 0 0 6px rgba(0,0,0,0.3);
      border-radius: 10px;
      background-color: transparent;
      margin-top: 10px;
      margin-bottom: 10px;
    }

    .tableFixed::-webkit-scrollbar
    {
      width: 12px;
      background-color: transparent;
    }

    .tableFixed::-webkit-scrollbar-thumb
    {
      border-radius: 10px;
      -webkit-box-shadow: inset 0 0 6px rgba(0,0,0,.3);
      background-color: #555;
    }
    
    .box h2 {
      font-size: 1.5rem;
      margin-top: 0;
      color: #00ed76;
    }
    
    .list {
      margin-top: 10px;
      width: 100%;
    }
    
    .tableFixed {
      width: 100%;
      border-collapse: collapse;
      margin-top: 10px;
      table-layout: fixed;
    }

    .tableFixed tbody {
      display: block;
      overflow-y: scroll;
      height: 375px;
    }

    .tableFixed thead tr {
      display: block;
    }
    
    th, td {
      padding: 5px;
      text-align: left;
    }
    
    th {
      border-bottom: 1px solid white;
    }
    
    .chart {
      margin-top: 20px;
      height: 70vh;
      display: flex;
      justify-content: center;
    }

    .finePrint {
        padding-top: 10px;
        display: flex;
        justify-content: center;
        color: #2d2e35;
    }

    .footer {
        margin-top: 80px;
        padding-bottom: 30px;
        padding-top: 30px;
        background-color: #18191a;
        display: flex;
        justify-content: center;
    }

  </style>
</head>
<body>
<div class="heading">
    <img src="https://pon-app.netlify.app/static/media/icon-builder.7fb453f0.svg" alt="icon" style="padding: 10px;">
    Blockswap PoN Builder
</div>

  
<div class="container">
    
    <div class="box">
    <h2>API Status</h2>
    <div class="list">
    <table>
    <tbody>
        <tr>
        <td>Geth Sync Status</td>
        <td>{{ .SyncStatus }}</td>
        </tr>
        <tr>
        <td>PoN Builder API Status</td>
        <td>OK</td>
        </tr>
        <tr>
        <td>PoN Relay Connected</td>
        <td>{{ .RelayUrl }}</td>
        </tr>
        <tr>
        <td>PoN Relay API Status</td>
        <td>{{ .RelayStatus }}</td>
        </tr>
        <tr>
        <td>PoN Builder Metrics Enabled</td>
        <td>{{ .MetricsEnabled }}</td>
        </tr>
        <tr>
        <td>Total bids sent</td>
        <td>{{ .TotalBidsSent }}</td>
        </tr>
        <tr>
        <td>Total bids won</td>
        <td>{{ .TotalBidsWon }}</td>
        </tr>
        <tr>
        <td>Total bids added to chain</td>
        <td>{{ .TotalBidsSubmittedToChain }}</td>
        </tr>
        <tr>
        <td>Total MEV of winning bids</td>
        <td>{{ .WinningBidsMEV }} ETH</td>
        </tr>
        <tr>
        <td>Total spent on winning bids</td>
        <td>{{ .WinningBidsBidAmount }} ETH</td>
        </tr>
    </tbody>
    </table>
    </div>
    </div>


    <div class="box">
      <h2>Recent Block Bids Sent</h2>
      <div class="list">
        <table class="tableFixed">
          <thead>
            <tr>
              <th>Slot</th>
              <th>Bid Amount (ETH)</th>
              <th>MEV (ETH)</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {{ range .RecentBids }}
            <tr>
              <td>{{ .Slot }}</td>
              <td>{{ .BidAmount }}</td>
              <td>{{ .MEV }}</td>
              <td>{{ .SentAt }}</td>
            </tr>
            {{ end }}
          </tbody>
        </table>
      </div>
    </div>
        <div class="box">
            <h2>Recent Block Bids Won</h2>
            <div class="list">
            <table class="tableFixed">
            <thead>
              <tr>
                <th>Slot</th>
                <th>Bid Amount (ETH)</th>
                <th>MEV (ETH)</th>
                <th>Date</th>
              </tr>
            </thead>
            <tbody>
              {{ range .RecentBidsWon }}
              <tr>
                <td>{{ .Slot }}</td>
                <td>{{ .BidAmount }}</td>
                <td>{{ .MEV }}</td>
                <td>{{ .SentAt }}</td>
              </tr>
              {{ end }}
            </tbody>
          </table>
            </div>
        </div>


    </div>

    <div class="containerFull">
    
    <div class="box">
      <h2>Performance</h2>
      <div class="chart">
      <canvas id="performanceChart"> </canvas>
      <script>
          var ctx = document.getElementById('performanceChart');
          var performanceChart = new Chart(ctx, {
              data: {
                  labels: [{{ range .ChartLabels }}'{{ . }}',{{ end}}],
                  datasets: [{
                    type: 'line',
                    label: 'Average Bid Amount (ETH)',
                    data: [{{ range .ChartAvgBidAmounts }}{{ . }},{{ end }}],
                    borderColor: 'rgba(255, 99, 132, 0.8)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)',
                    borderDash: [10, 5],
                    yAxisID: 'yLine'
                  },{
                    type: 'line',
                    label: 'Average Block MEV (ETH)',
                    data: [{{ range .ChartAvgMEVs }}{{ . }},{{ end }}],
                    borderColor: 'rgba(50, 168, 52, 0.8)',
                    backgroundColor: 'rgba(50, 168, 52, 0.2)',
                    borderDash: [10, 5],
                    yAxisID: 'yLine'
                  },{
                    type: 'line',
                    label: 'Total Spent on Winning Bids (ETH)',
                    data: [{{ range .ChartTotalBidsWonBidAmounts }}{{ . }},{{ end }}],
                    borderColor: 'rgba(255, 99, 132, 1)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)',
                    yAxisID: 'yLine'
                  },{
                    type: 'line',
                    label: 'Winning Blocks Total MEV (ETH)',
                    data: [{{ range .ChartTotalBidsWonMEVs }}{{ . }},{{ end }}],
                    borderColor: 'rgba(50, 168, 52, 1)',
                    backgroundColor: 'rgba(50, 168, 52, 0.2)',
                    yAxisID: 'yLine'
                  },{
                    type: 'bar',
                    label: 'Total Bids Sent',
                    data: [{{ range .ChartBidsSent }}{{ . }},{{ end }}],
                    backgroundColor: 'rgba(168, 131, 50, 0.5)',
                    yAxisID: 'yBar'
                  },{
                    type: 'bar',
                    label: 'Total Bids Won',
                    data: [{{ range .ChartBidsWon }}{{ . }},{{ end }}],
                    backgroundColor: 'rgba(50, 50, 168, 0.5)',
                    yAxisID: 'yBar'
                  }]
              },
              options: {
                responsive: true,
                scales: {
                    yLine: {
                      type: 'linear',
                      display: true,
                      position: 'left',
                      beginAtZero: true,
                    },
                    yBar: {
                      type: 'linear',
                      display: true,
                      position: 'right',
                      beginAtZero: true,
              
                      grid: {
                        drawOnChartArea: false,
                      },
                    },
                }
              }
          });
      </script>
      </div>
    </div>

</div>



    <div class="containerFull">
    <div class="box">
      <h2>All Block Bids Sent</h2>
      <div class="list">
      <table class="tableFixed">
      <thead>
        <tr>
          <th>Slot</th>
          <th>Bid Amount (ETH)</th>
          <th>MEV (ETH)</th>
          <th>BlockHash</th>
          <th>Priority Transaction Count</th>
          <th>Total Transaction Count</th>
          <th>Date</th>
        </tr>
      </thead>
      <tbody>
        {{ range .AllBids }}
        <tr>
          <td>{{ .Slot }}</td>
          <td>{{ .BidAmount }}</td>
          <td>{{ .MEV }}</td>
          <td>{{ .BlockHash }}</td>
          <td>{{ .PrivateTxCount }}</td>
          <td>{{ .TotalTxCount }}</td>
          <td>{{ .SentAt }}</td>
        </tr>
        {{ end }}
      </tbody>
    </table>
      </div>
    </div>
    
  </div>

  <div class="finePrint">
    --- Bid history is only available for the last 1095750 slots (max 5 months of bid data) ---
  </div>

  <div class="footer">
    Copyright © 2023 Blockswap Labs. All Rights Reserved.
  </div>

  
</body>

</html>
      

`)
}
