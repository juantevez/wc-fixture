package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/team-registry/internal/domain/ports"
	"github.com/wc-fixture/team-registry/internal/domain/team"
)

// teamRepo implementa ports.TeamRepository.
type teamRepo struct {
	pool *pgxpool.Pool
}

var _ ports.TeamRepository = (*teamRepo)(nil)

func NewTeamRepository(pool *pgxpool.Pool) ports.TeamRepository {
	return &teamRepo{pool: pool}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *teamRepo) GetByID(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	const q = `
		SELECT id, name, short_name, country_code,
			   confederation, fifa_ranking_date, flag_url, qualified
		FROM teams WHERE id = $1`

	t, err := r.scanTeam(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("equipo", id.String())
	}
	if err != nil {
		return nil, fmt.Errorf("team_repo: error consultando equipo: %w", err)
	}
	return t, nil
}

// ── GetByShortName ────────────────────────────────────────────────────────────

func (r *teamRepo) GetByShortName(ctx context.Context, shortName string) (*team.Team, error) {
	const q = `
		SELECT id, name, short_name, country_code,
			   confederation, fifa_ranking_date, flag_url, qualified
		FROM teams WHERE UPPER(short_name) = UPPER($1)`

	t, err := r.scanTeam(r.pool.QueryRow(ctx, q, shortName))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("equipo", shortName)
	}
	if err != nil {
		return nil, fmt.Errorf("team_repo: error consultando equipo por short_name: %w", err)
	}
	return t, nil
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *teamRepo) List(ctx context.Context, filters ports.TeamFilters) ([]team.Team, error) {
	conditions := []string{"1=1"}
	args := []any{}
	n := 1

	if filters.Confederation != "" {
		conditions = append(conditions, fmt.Sprintf("confederation = $%d", n))
		args = append(args, string(filters.Confederation))
		n++
	}
	if filters.QualifiedOnly {
		conditions = append(conditions, "qualified = true")
	}

	q := fmt.Sprintf(`
		SELECT id, name, short_name, country_code,
			   confederation, fifa_ranking_date, flag_url, qualified
		FROM teams
		WHERE %s
		ORDER BY confederation, fifa_ranking_date ASC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("team_repo: error listando equipos: %w", err)
	}
	defer rows.Close()

	var teams []team.Team
	for rows.Next() {
		t, err := r.scanTeam(rows)
		if err != nil {
			return nil, err
		}
		teams = append(teams, *t)
	}
	return teams, rows.Err()
}

// ── Save ──────────────────────────────────────────────────────────────────────

func (r *teamRepo) Save(ctx context.Context, t team.Team) error {
	const q = `
		INSERT INTO teams
			(id, name, short_name, country_code,
			 confederation, fifa_ranking_date, flag_url, qualified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name               = EXCLUDED.name,
			short_name         = EXCLUDED.short_name,
			country_code       = EXCLUDED.country_code,
			confederation      = EXCLUDED.confederation,
			fifa_ranking_date  = EXCLUDED.fifa_ranking_date,
			flag_url           = EXCLUDED.flag_url,
			qualified          = EXCLUDED.qualified,
			updated_at         = NOW()`

	if _, err := r.pool.Exec(ctx, q,
		t.ID, t.Name, t.ShortName, t.CountryCode,
		string(t.Confederation), t.FIFARankingDate, t.FlagURL, t.Qualified,
	); err != nil {
		return fmt.Errorf("team_repo: error guardando equipo %s: %w", t.ID, err)
	}
	return nil
}

// ── ListConfederations ────────────────────────────────────────────────────────

func (r *teamRepo) ListConfederations(ctx context.Context) ([]team.Confederation, error) {
	const q = `
		SELECT code, name, short_name, region
		FROM confederations
		ORDER BY code`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("team_repo: error listando confederaciones: %w", err)
	}
	defer rows.Close()

	var confederations []team.Confederation
	for rows.Next() {
		var c team.Confederation
		var codeStr string
		if err := rows.Scan(&codeStr, &c.Name, &c.ShortName, &c.Region); err != nil {
			return nil, fmt.Errorf("team_repo: error escaneando confederación: %w", err)
		}
		c.Code = team.ConfederationCode(codeStr)
		confederations = append(confederations, c)
	}
	return confederations, rows.Err()
}

// ── Helper de scan ────────────────────────────────────────────────────────────

func (r *teamRepo) scanTeam(scanner interface {
	Scan(dest ...any) error
}) (*team.Team, error) {
	var t team.Team
	var confederationStr string

	if err := scanner.Scan(
		&t.ID, &t.Name, &t.ShortName, &t.CountryCode,
		&confederationStr, &t.FIFARankingDate, &t.FlagURL, &t.Qualified,
	); err != nil {
		return nil, err
	}

	t.Confederation = team.ConfederationCode(confederationStr)
	return &t, nil
}
