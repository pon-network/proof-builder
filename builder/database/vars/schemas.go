package vars

var ForceDropSchema = `
DROP TABLE IF EXISTS blockbid;
DROP TABLE IF EXISTS blindedbeaconblock;
DROP TABLE IF EXISTS beaconblock;
`

var DropSchema = `
BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='blockbid';

SELECT
	CASE 
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS blockbid;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*) 
			FROM pragma_table_info('blockbid')
			WHERE name IN (
				'id', 'inserted_at', 'signature', 'slot', 'builder_pubkey', 'proposer_pubkey', 'fee_recipient', 'builder_wallet_address', 'gas_used', 'gas_limit', 'mev', 'payout_pool_tx', 'payout_pool_address', 'payout_pool_gas_fee', 'rpbs', 'priority_transactions_count', 'transactions_count', 'block_hash', 'parent_hash', 'block_number', 'value'
			)
		) = 21 THEN NULL
		ELSE 'DROP TABLE IF EXISTS blockbid;'
	END
FROM sqlite_master
WHERE type='table' AND name='blockbid';

COMMIT;

BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='blindedbeaconblock';

SELECT
	CASE
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS blindedbeaconblock;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*)
			FROM pragma_table_info('blindedbeaconblock')
			WHERE name IN (
				'id', 'inserted_at', 'bid_id', 'signed_blinded_beacon_block', 'signature'
			)
		) = 5 THEN NULL
		ELSE 'DROP TABLE IF EXISTS blindedbeaconblock;'
	END
FROM sqlite_master
WHERE type='table' AND name='blindedbeaconblock';

COMMIT;

BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='beaconblock';

SELECT
	CASE
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS beaconblock;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*)
			FROM pragma_table_info('beaconblock')
			WHERE name IN (
				'id', 'inserted_at', 'bid_id', 'signed_beacon_block', 'signature', 'submitted_to_chain', 'submitted_to_chain_at', 'submitted_to_chain_error'
			)
		) = 7 THEN NULL

		ELSE 'DROP TABLE IF EXISTS beaconblock;'
	END
FROM sqlite_master
WHERE type='table' AND name='beaconblock';

COMMIT;
`

var CreateSchema = `

CREATE TABLE IF NOT EXISTS blockbid (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	signature TEXT NOT NULL,

	slot INTEGER NOT NULL,

	builder_pubkey TEXT NOT NULL,
	proposer_pubkey TEXT NOT NULL,
	fee_recipient TEXT NOT NULL,
	builder_wallet_address TEXT NOT NULL,

	gas_used INTEGER NOT NULL,
	gas_limit INTEGER NOT NULL,

	mev INTEGER NOT NULL,

	payout_pool_tx TEXT NOT NULL,
	payout_pool_address TEXT NOT NULL,
	payout_pool_gas_fee INTEGER NOT NULL,
	rpbs TEXT NOT NULL,	

	priority_transactions_count INTEGER NOT NULL,
	transactions_count INTEGER NOT NULL,

	block_hash TEXT NOT NULL,
	parent_hash TEXT NOT NULL,
	block_number INTEGER NOT NULL,

	value INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS blindedbeaconblock (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	bid_id INTEGER NOT NULL,

	signed_blinded_beacon_block TEXT NOT NULL,

	signature TEXT NOT NULL,

	FOREIGN KEY(bid_id) REFERENCES blockbid(id)
);

CREATE TABLE IF NOT EXISTS beaconblock (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	bid_id INTEGER NOT NULL,

	signed_beacon_block TEXT NOT NULL,

	signature TEXT NOT NULL,

	submitted_to_chain BOOLEAN NOT NULL,
	submitted_to_chain_at TIMESTAMP NULL,
	submitted_to_chain_error TEXT NULL,

	FOREIGN KEY(bid_id) REFERENCES blockbid(id)
);

`
