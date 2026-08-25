package infrastructure

const ClaimJobSQL = `
UPDATE jobs SET status = 'running', locked_at = now()
WHERE id = (
  SELECT id FROM jobs WHERE status = 'pending' AND run_after <= now()
  ORDER BY run_after, created_at FOR UPDATE SKIP LOCKED LIMIT 1
)
RETURNING id, kind, reference_id, attempt, max_attempts, run_after, created_at`

const CompleteJobSQL = `UPDATE jobs SET status = 'completed', completed_at = now() WHERE id = $1`

const RetryJobSQL = `UPDATE jobs SET status = 'pending', attempt = $2, run_after = $3, last_error = $4 WHERE id = $1`
