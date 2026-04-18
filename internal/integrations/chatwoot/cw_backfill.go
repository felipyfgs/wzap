package chatwoot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wzap/internal/logger"
)

// BackfillResult summarizes the outcome of a BackfillCloudRefs run.
type BackfillResult struct {
	Scanned  int `json:"scanned"`
	Updated  int `json:"updated"`
	NotFound int `json:"notFound"`
}

// ErrBackfillUnavailable is returned when the Chatwoot configuration does not
// expose a database_uri, making the direct lookup strategy unusable.
var ErrBackfillUnavailable = errors.New("chatwoot database_uri is not configured")

// BackfillCloudRefs walks wz_messages rows for the given session that still
// lack Chatwoot references and tries to resolve them via a direct read-only
// query on the Chatwoot database using the configured database_uri. Every
// match is written back to wz_messages through the regular
// UpdateChatwootRef repository call so downstream consumers (reply/edit/
// revoke) can use the cached references.
func (s *Service) BackfillCloudRefs(ctx context.Context, sessionID string) (BackfillResult, error) {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("load chatwoot config: %w", err)
	}
	if cfg.DatabaseURI == "" {
		return BackfillResult{}, ErrBackfillUnavailable
	}

	pool, err := getPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("chatwoot mapping pool: %w", err)
	}

	const batchSize = 500
	var (
		result  BackfillResult
		afterID string
		start   = time.Now()
	)

	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		ids, err := s.msgRepo.ListMissingChatwootRefs(ctx, sessionID, batchSize, afterID)
		if err != nil {
			return result, fmt.Errorf("list missing refs: %w", err)
		}
		if len(ids) == 0 {
			break
		}
		result.Scanned += len(ids)

		sourceIDs := make([]string, len(ids))
		for i, id := range ids {
			sourceIDs[i] = "WAID:" + id
		}

		rows, err := pool.Query(ctx,
			`SELECT source_id, id, conversation_id
			   FROM messages
			  WHERE account_id = $1
			    AND source_id = ANY($2::text[])`,
			cfg.AccountID, sourceIDs)
		if err != nil {
			return result, fmt.Errorf("query chatwoot messages: %w", err)
		}

		type match struct {
			waMsgID  string
			cwMsgID  int
			cwConvID int
			sourceID string
		}
		matches := make([]match, 0, len(ids))
		for rows.Next() {
			var m match
			if err := rows.Scan(&m.sourceID, &m.cwMsgID, &m.cwConvID); err != nil {
				rows.Close()
				return result, fmt.Errorf("scan chatwoot match: %w", err)
			}
			if len(m.sourceID) > 5 && m.sourceID[:5] == "WAID:" {
				m.waMsgID = m.sourceID[5:]
			} else {
				m.waMsgID = m.sourceID
			}
			matches = append(matches, m)
		}
		rowsErr := rows.Err()
		rows.Close()
		if rowsErr != nil {
			return result, fmt.Errorf("iterate chatwoot matches: %w", rowsErr)
		}

		for _, m := range matches {
			if err := s.msgRepo.UpdateChatwootRef(ctx, sessionID, m.waMsgID, m.cwMsgID, m.cwConvID, m.sourceID); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("waMsgID", m.waMsgID).Int("cwMsgID", m.cwMsgID).Msg("backfill: failed to update chatwoot ref")
				continue
			}
			result.Updated++
		}
		result.NotFound += len(ids) - len(matches)

		afterID = ids[len(ids)-1]
		if len(ids) < batchSize {
			break
		}
	}

	logger.Info().Str("component", "chatwoot").Str("session", sessionID).Int("scanned", result.Scanned).Int("updated", result.Updated).Int("notFound", result.NotFound).Dur("duration", time.Since(start)).Msg("chatwoot cloud refs backfill finished")
	return result, nil
}
