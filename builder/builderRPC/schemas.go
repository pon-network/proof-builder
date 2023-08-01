package builderRPC

var DropSchema = `
BEGIN TRANSACTION;

SELECT COUNT(*) as count FROM sqlite_master WHERE type='table' AND name='rpctxs';

SELECT
	CASE
		WHEN COUNT(*) = 0 THEN 'DROP TABLE IF EXISTS rpctxs;'
		WHEN COUNT(*) = 1 AND (
			SELECT COUNT(*)
			FROM pragma_table_info('rpctxs')
			WHERE name IN (
				'id', 'inserted_at', 'tx_hash'
			)
		) = 3 THEN NULL
		ELSE 'DROP TABLE IF EXISTS rpctxs;'
	END
FROM sqlite_master
WHERE type='table' AND name='rpctxs';

COMMIT;

`

var CreateSchema = `

CREATE TABLE IF NOT EXISTS rpctxs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	tx_hash TEXT NOT NULL
);

`
