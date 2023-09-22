package beaconClient

import (
	"net/url"
)

type beaconClient struct {
	beaconEndpoint *url.URL
}

func NewBeaconClient(endpoint string) (*beaconClient, error) {
	/*
		Initializes a new beacon client
	*/
	u, err := url.Parse(endpoint)
	bc := &beaconClient{
		beaconEndpoint: u,
	}

	// Client is initialized by multiBeacon client.
	// client instances and errors are handled by multiBeacon client.
	return bc, err
}

func (b *beaconClient) BaseEndpoint() string {
	return b.beaconEndpoint.String()
}
