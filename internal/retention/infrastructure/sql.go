package infrastructure

const DeleteExpiredRawSQL = `DELETE FROM hl7_messages WHERE id IN (SELECT id FROM hl7_messages WHERE raw_expires_at < $1 LIMIT $2)`
const DeleteExpiredMetadataSQL = `DELETE FROM hl7_messages WHERE id IN (SELECT id FROM hl7_messages WHERE metadata_expires_at < $1 LIMIT $2)`
const DeleteExpiredDeadLetterSQL = `DELETE FROM dead_letters WHERE id IN (SELECT id FROM dead_letters WHERE expires_at < $1 LIMIT $2)`
