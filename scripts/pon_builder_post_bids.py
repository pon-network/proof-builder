import requests
import json
import time
import paho.mqtt.client as mqtt
from collections import defaultdict
from multiprocessing import Process
from sseclient import SSEClient


class BlockBidder:
    """
    BlockBidder for PoN builder. This class will submit bids to the builder
    for the current slot. If the bid is successful, the block will be built
    and submitted to the relay. If the bid is unsuccessful, the block will
    not be built and the bid will be discarded.

    There is the bulletin board that can be used to check the status of the
    bids, and bid value for slots, this can be connected too using the
    designated broker and topics.
    """

    def __init__(
        self,
        bid_url,
        beacon_url,
        default_bid,
        fee_recipient,
        auto_slot=True,
        mqtt_broker=""
    ):
        self.bid_url = bid_url
        self.beacon_url = beacon_url
        self.slot_count = 0
        self.slot_bid_amount = defaultdict(lambda: default_bid)
        self.fee_recipient = fee_recipient
        self.transactions = []
        self.no_mempool_txs = "false"
        self.auto_slot = auto_slot
        self.enable_bulletin_board = mqtt_broker != ""
        self.mqtt_broker = mqtt_broker
        self.client = None

        self.wl_bid_addresses = []
        # Whitelisted bid addresses that are allowed
        # to keep current highest bid without BlockBidder
        # attempting to outbid them

        # self.mu = Semaphore(1)

    def set_wl_bid_addresses(self, wl_bid_addresses):
        """
        The function sets the whitelisted bid addresses.

        :param wl_bid_addresses: The wl_bid_addresses parameter is used to set
        the whitelisted bid addresses.
        """
        addresses = []
        for address in wl_bid_addresses:
            address = address.strip().lower()
            addresses.append(address)
        self.wl_bid_addresses = addresses

    def set_bid_amount(self, slot, bid_amount):
        """
        The function sets the default bid amount for a specific slot.

        :param slot: The slot parameter is used to set the default bid amount
        for a specific slot.
        :param bid_amount: The bid_amount parameter is used to set the default
        bid amount for a specific slot.
        """
        self.slot_bid_amount[slot] = bid_amount

    def connect_bulletin_board(self):
        """
        The function connects to the bulletin board.
        """
        if self.mqtt_broker is None:
            raise Exception("No MQTT broker specified")
        self.client = mqtt.Client()
        self.client.connect(self.mqtt_broker)
        self.client.loop_start()

    def subscribe(self):
        """
        The function subscribes to the bulletin board.
        """
        self.client.subscribe(
            ("topic/HighestBid", 0),
            ("topic/ProposerSlotHeaderRequest", 0),
            ("topic/ProposerPayloadRequest", 0),
        )
        self.client.on_message = self.on_message

    def on_message(self, client, userdata, message):
        """
        The `on_message` function prints information about a new message
        received from a bulletin board
        and performs actions based on the message content.

        :param client: The `client` parameter is an instance of the MQTT
        client that is used to connect
        to the MQTT broker and publish/subscribe to topics
        :param userdata: The `userdata` parameter is a user-defined data that
        can be passed to the
        `on_message` function. It can be used to store any additional
        information or context that you
        want to access within the function. In this code snippet, the
        `userdata` parameter is not used,
        so it can be
        :param message: The `message` parameter is the MQTT message
        received by the client. It contains
        information such as the topic of the message and the payload
        (message content)
        :return: The function does not explicitly return anything.
        """
        # print("\n****************************\n")
        # print("Bulletin Board new message:")
        # print("Topic:", message.topic)
        # print("Message:", str(message.payload.decode("utf-8")))
        # print("\n****************************\n")

        if message.topic == "topic/HighestBid":
            # message is a comma separated string of the form:
            # slot, address, bid_amount
            # e.g. slot: 0, address: 0x123, bid_amount: 1000000000000000000
            data = message.payload.decode("utf-8").split(
                ","
            )
            slot, address, bid_amount = data[0], data[1], data[2]
            slot = slot.split(":")[1].strip()
            address = address.split(":")[1].strip()
            bid_amount = bid_amount.split(":")[1].strip()

            if int(slot) > int(self.slot_count):
                self.slot_count = int(slot)

            print("\n****************************\n")
            print("Slot bid for: " + str(slot))
            print("Highest bid address: " + address)
            print("Current bid amount: " + str(bid_amount))

            if str(address).strip().lower() in self.wl_bid_addresses:
                print("\nDo not attempt to outbid")
                print("Whitelisted address:", address)
                print("\n****************************\n")
                return

            print("Attempting to outbid...")
            print("Sending bid amount:",
                  str(int(bid_amount)+1))
            print("\n****************************\n")

            # self.place_bid(int(slot), int(bid_amount)+1)

    def place_bid(self, slot=None, bid_amount=None):
        """
        The function is used to submit a bid for a specific slot.
        """

        payload = {
            "suggestedFeeRecipient": self.fee_recipient,
            "transactions": self.transactions,
            "noMempoolTxs": self.no_mempool_txs
        }

        if slot is None:
            slot = self.slot_count
        else:
            payload["slot"] = str(slot)

        if not self.auto_slot:
            payload["slot"] = str(slot)

        if bid_amount is None:
            bid_amount = self.slot_bid_amount[slot]

        payload["bidAmount"] = str(bid_amount)

        correct_slot = False

        headers = {
            'Content-Type': 'application/json'
        }

        response = requests.request(
            "POST",
            self.bid_url,
            headers=headers,
            data=json.dumps(payload)
        )

        if response.status_code != 200:
            if "slot for bid" in response.text:
                current_slot = response.text.split(
                    "next available slot for bid is "
                )[1].replace('"}\n', "")

                print("Error submitting bid for slot:", slot)
                print("Error:", response.text)

                if int(current_slot) > int(slot):
                    slot = int(current_slot)

                correct_slot = True

        if response.status_code == 200:
            print("------------------------------\n")
            json_response = json.loads(response.text)

            current_user_bid = None
            self.slot_bid_amount[slot] = bid_amount

            if isinstance(json_response, list):
                if len(json_response) > 0:
                    current_user_bid = json_response[-1]

                    try:
                        slot_count = int(
                            current_user_bid["block_bid"]["message"]["slot"])
                        print(
                            "Successfully submitted bid for slot:",
                            str(slot_count)
                        )

                        print("")

                        print("Finalized bid: ")
                        print(current_user_bid["block_bid"])
                        print("")
                        print("Relay response: ")
                        print(current_user_bid["relay_response"])
                        print("")
                        print("Bid requested at: ")
                        print(current_user_bid["bid_request_time"])
                        print("")
                        print("Block built at: ")
                        print(current_user_bid["block_built_time"])
                        print("")
                        print("Block submitted at: ")
                        print(current_user_bid["block_submitted_time"])
                        print("")
                        print("Bids sent to relay: ")
                        print(len(json_response))
                        print("")
                        print("All submitted bids: ")
                        print(json_response)

                    except KeyError:
                        print("Successfully submitted bid for current slot")

                        print("")

                        print(json_response)

                else:
                    print("No bids submitted. Builder failed without error.")
                    print(json_response)
            else:
                print(json_response)

            print("\n------------------------------\n")
        elif (
            ("duplicate" not in response.text) and
            ("slot for bid" not in response.text)
        ):
            print("Error submitting bid for slot:", slot)
            print("Error:", response.text)

        if correct_slot:
            slot += 1

        if slot > self.slot_count:
            self.slot_count = slot

    def submit_bids(self):
        """
        The function is used to submit bids for all slots.
        """
        messages = SSEClient(self.beacon_url + "/eth/v1/events?topics=head")

        for event in messages:
            try:
                print("Head event received")
                print(event.data)

                slot = int(json.loads(event.data)["slot"])
                self.place_bid()
            except Exception as submission_error:
                print("Error submitting bid for slot from head event")
                print(submission_error)

    def start(self):
        """
        The function is used to start the block bidder.
        """
        print("Starting block bidder\n")
        if self.enable_bulletin_board:
            self.connect_bulletin_board()
            self.subscribe()

        # use multi-processing to submit bids for all slots
        p = Process(target=self.submit_bids)
        p.start()


if __name__ == "__main__":

    BLOCK_BUILDER_ADDRESS = "0x22A3864baaE65a9e8E5C163F80F850ADFe40Ed90"
    BEACON_NODE_URL = "http://localhost:3001"
    BUILDER_BID_URL = "http://localhost:10000/eth/v1/builder/submit_block_bid"
    BROKER = "broker.emqx.io"

    # Bid amount is in wei, so 0.04 ETH = 40000000000000000 wei
    DEFAULT_BID = 40000000000000000  # 0.04 ETH

    FEE_RECIPIENT = "0x22A3864baaE65a9e8E5C163F80F850ADFe40Ed90"

    # These are a set of addresses defined
    # for which the block bidder should not attempt to outbid
    # upon receiving a a bulletin board message
    # about there being a new current highest bidder
    WL_BID_ADDRESSES = [
        BLOCK_BUILDER_ADDRESS
    ]

    bidder = BlockBidder(
        BUILDER_BID_URL,
        BEACON_NODE_URL,
        DEFAULT_BID,
        FEE_RECIPIENT,
        auto_slot=True,
        mqtt_broker=BROKER
    )

    bidder.set_wl_bid_addresses(WL_BID_ADDRESSES)

    bidder.start()
