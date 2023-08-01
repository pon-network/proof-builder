package bundles

var ForceDropSchema = `
DROP TABLE IF EXISTS blockbundles;
DROP TABLE IF EXISTS sentbundles;
`

var DropSchema = `
BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='blockbundles';

SELECT
	CASE 
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS blockbundles;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*) 
			FROM pragma_table_info('blockbundles')
			WHERE name IN (
				'id', 'inserted_at', 'bundle_hash', 'txs', 'block_number', 'min_timestamp', 'max_timestamp', 'reverting_tx_hashes', 'builder_pubkey', 'builder_signature', 'bundle_transaction_count', 'bundle_total_gas', 'added', 'error', 'error_message', 'failed_retry_count'
			)
		) = 16 THEN NULL
		ELSE 'DROP TABLE IF EXISTS blockbundles;'
	END
FROM sqlite_master
WHERE type='table' AND name='blockbundles';

COMMIT;

BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='sentbundles';

SELECT
	CASE
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS sentbundles;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*)
			FROM pragma_table_info('sentbundles')
			WHERE name IN (
				'id', 'bid_id', 'bundle_hash', 'slot'
			)
		) = 4 THEN NULL
		ELSE 'DROP TABLE IF EXISTS sentbundles;'
	END
FROM sqlite_master
WHERE type='table' AND name='sentbundles';

COMMIT;

`

var CreateSchema = `

CREATE TABLE IF NOT EXISTS blockbundles (
	id STRING PRIMARY KEY,
	inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	bundle_hash TEXT NOT NULL,
	txs TEXT NOT NULL,
	block_number INTEGER NOT NULL,
	min_timestamp INTEGER NOT NULL,
	max_timestamp INTEGER NOT NULL,
	reverting_tx_hashes TEXT NOT NULL,

	builder_pubkey TEXT NOT NULL,
	builder_signature TEXT NOT NULL,

	bundle_transaction_count INTEGER NOT NULL,
	bundle_total_gas INTEGER NOT NULL,

	added INTEGER NOT NULL DEFAULT 0,
	error INTEGER NOT NULL DEFAULT 0,

	failed_retry_count INTEGER NOT NULL DEFAULT 0,

	error_message TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sentbundles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,

	block_hash TEXT NOT NULL,

	slot INTEGER NOT NULL,

	bundle_hash TEXT NOT NULL

);

`
