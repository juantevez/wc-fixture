-- =============================================================================
-- seed_teams.sql — 48 selecciones clasificadas al Mundial 2026
-- Ejecutar contra team_db después de correr las migraciones.
-- Uso: psql -U wc2026 -d team_db -f seed_teams.sql
--
-- Fuente: clasificación oficial FIFA Mundial 2026
-- Ranking FIFA: posiciones al momento del sorteo (diciembre 2025, estimado)
-- =============================================================================

-- Limpiar datos previos (idempotente)
TRUNCATE TABLE teams RESTART IDENTITY CASCADE;

INSERT INTO teams (id, name, short_name, country_code, confederation, fifa_ranking_date, flag_url, qualified)
VALUES

-- ═══════════════════════════════════════════════════════════════════════════
-- UEFA — 16 cupos (Europa)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000001', 'España',          'ESP', 'ESP', 'UEFA',     1,  'https://flagcdn.com/es.svg', TRUE),
('00000000-0000-0000-0001-000000000002', 'Francia',         'FRA', 'FRA', 'UEFA',     2,  'https://flagcdn.com/fr.svg', TRUE),
('00000000-0000-0000-0001-000000000003', 'Inglaterra',      'ENG', 'ENG', 'UEFA',     3,  'https://flagcdn.com/gb-eng.svg', TRUE),
('00000000-0000-0000-0001-000000000004', 'Alemania',        'GER', 'DEU', 'UEFA',     4,  'https://flagcdn.com/de.svg', TRUE),
('00000000-0000-0000-0001-000000000005', 'Portugal',        'POR', 'PRT', 'UEFA',     6,  'https://flagcdn.com/pt.svg', TRUE),
('00000000-0000-0000-0001-000000000006', 'Países Bajos',    'NED', 'NLD', 'UEFA',     7,  'https://flagcdn.com/nl.svg', TRUE),
('00000000-0000-0000-0001-000000000007', 'Bélgica',         'BEL', 'BEL', 'UEFA',     8,  'https://flagcdn.com/be.svg', TRUE),
('00000000-0000-0000-0001-000000000008', 'Italia',          'ITA', 'ITA', 'UEFA',     9,  'https://flagcdn.com/it.svg', TRUE),
('00000000-0000-0000-0001-000000000009', 'Croacia',         'CRO', 'HRV', 'UEFA',     10, 'https://flagcdn.com/hr.svg', TRUE),
('00000000-0000-0000-0001-000000000010', 'Dinamarca',       'DEN', 'DNK', 'UEFA',     11, 'https://flagcdn.com/dk.svg', TRUE),
('00000000-0000-0000-0001-000000000011', 'Austria',         'AUT', 'AUT', 'UEFA',     12, 'https://flagcdn.com/at.svg', TRUE),
('00000000-0000-0000-0001-000000000012', 'Suiza',           'SUI', 'CHE', 'UEFA',     13, 'https://flagcdn.com/ch.svg', TRUE),
('00000000-0000-0000-0001-000000000013', 'Turquía',         'TUR', 'TUR', 'UEFA',     18, 'https://flagcdn.com/tr.svg', TRUE),
('00000000-0000-0000-0001-000000000014', 'Serbia',          'SRB', 'SRB', 'UEFA',     20, 'https://flagcdn.com/rs.svg', TRUE),
('00000000-0000-0000-0001-000000000015', 'Polonia',         'POL', 'POL', 'UEFA',     22, 'https://flagcdn.com/pl.svg', TRUE),
('00000000-0000-0000-0001-000000000016', 'Escocia',         'SCO', 'SCO', 'UEFA',     25, 'https://flagcdn.com/gb-sct.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- CONMEBOL — 6 cupos (Sudamérica)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000017', 'Argentina',       'ARG', 'ARG', 'CONMEBOL',  5,  'https://flagcdn.com/ar.svg', TRUE),
('00000000-0000-0000-0001-000000000018', 'Brasil',          'BRA', 'BRA', 'CONMEBOL',  14, 'https://flagcdn.com/br.svg', TRUE),
('00000000-0000-0000-0001-000000000019', 'Colombia',        'COL', 'COL', 'CONMEBOL',  15, 'https://flagcdn.com/co.svg', TRUE),
('00000000-0000-0000-0001-000000000020', 'Uruguay',         'URU', 'URY', 'CONMEBOL',  16, 'https://flagcdn.com/uy.svg', TRUE),
('00000000-0000-0000-0001-000000000021', 'Ecuador',         'ECU', 'ECU', 'CONMEBOL',  27, 'https://flagcdn.com/ec.svg', TRUE),
('00000000-0000-0000-0001-000000000022', 'Venezuela',       'VEN', 'VEN', 'CONMEBOL',  30, 'https://flagcdn.com/ve.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- CONCACAF — 6 cupos (Norte/Centro América y Caribe)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000023', 'México',          'MEX', 'MEX', 'CONCACAF',  17, 'https://flagcdn.com/mx.svg', TRUE),
('00000000-0000-0000-0001-000000000024', 'Estados Unidos',  'USA', 'USA', 'CONCACAF',  19, 'https://flagcdn.com/us.svg', TRUE),
('00000000-0000-0000-0001-000000000025', 'Canadá',          'CAN', 'CAN', 'CONCACAF',  21, 'https://flagcdn.com/ca.svg', TRUE),
('00000000-0000-0000-0001-000000000026', 'Costa Rica',      'CRC', 'CRI', 'CONCACAF',  40, 'https://flagcdn.com/cr.svg', TRUE),
('00000000-0000-0000-0001-000000000027', 'Panamá',          'PAN', 'PAN', 'CONCACAF',  50, 'https://flagcdn.com/pa.svg', TRUE),
('00000000-0000-0000-0001-000000000028', 'Honduras',        'HON', 'HND', 'CONCACAF',  60, 'https://flagcdn.com/hn.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- CAF — 9 cupos (África)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000029', 'Marruecos',       'MAR', 'MAR', 'CAF',       13, 'https://flagcdn.com/ma.svg', TRUE),
('00000000-0000-0000-0001-000000000030', 'Senegal',         'SEN', 'SEN', 'CAF',       23, 'https://flagcdn.com/sn.svg', TRUE),
('00000000-0000-0000-0001-000000000031', 'Egipto',          'EGY', 'EGY', 'CAF',       32, 'https://flagcdn.com/eg.svg', TRUE),
('00000000-0000-0000-0001-000000000032', 'Nigeria',         'NGA', 'NGA', 'CAF',       34, 'https://flagcdn.com/ng.svg', TRUE),
('00000000-0000-0000-0001-000000000033', 'Costa de Marfil', 'CIV', 'CIV', 'CAF',       37, 'https://flagcdn.com/ci.svg', TRUE),
('00000000-0000-0000-0001-000000000034', 'Argelia',         'ALG', 'DZA', 'CAF',       39, 'https://flagcdn.com/dz.svg', TRUE),
('00000000-0000-0000-0001-000000000035', 'Sudáfrica',       'RSA', 'ZAF', 'CAF',       55, 'https://flagcdn.com/za.svg', TRUE),
('00000000-0000-0000-0001-000000000036', 'Túnez',           'TUN', 'TUN', 'CAF',       58, 'https://flagcdn.com/tn.svg', TRUE),
('00000000-0000-0000-0001-000000000037', 'Camerún',         'CMR', 'CMR', 'CAF',       61, 'https://flagcdn.com/cm.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- AFC — 8 cupos (Asia)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000038', 'Japón',           'JPN', 'JPN', 'AFC',       26, 'https://flagcdn.com/jp.svg', TRUE),
('00000000-0000-0000-0001-000000000039', 'Corea del Sur',   'KOR', 'KOR', 'AFC',       28, 'https://flagcdn.com/kr.svg', TRUE),
('00000000-0000-0000-0001-000000000040', 'Irán',            'IRN', 'IRN', 'AFC',       29, 'https://flagcdn.com/ir.svg', TRUE),
('00000000-0000-0000-0001-000000000041', 'Australia',       'AUS', 'AUS', 'AFC',       31, 'https://flagcdn.com/au.svg', TRUE),
('00000000-0000-0000-0001-000000000042', 'Arabia Saudita',  'KSA', 'SAU', 'AFC',       33, 'https://flagcdn.com/sa.svg', TRUE),
('00000000-0000-0000-0001-000000000043', 'Catar',           'QAT', 'QAT', 'AFC',       35, 'https://flagcdn.com/qa.svg', TRUE),
('00000000-0000-0000-0001-000000000044', 'Uzbekistán',      'UZB', 'UZB', 'AFC',       67, 'https://flagcdn.com/uz.svg', TRUE),
('00000000-0000-0000-0001-000000000045', 'Irak',            'IRQ', 'IRQ', 'AFC',       72, 'https://flagcdn.com/iq.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- OFC — 1 cupo (Oceanía)
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000046', 'Nueva Zelanda',   'NZL', 'NZL', 'OFC',       98, 'https://flagcdn.com/nz.svg', TRUE),

-- ═══════════════════════════════════════════════════════════════════════════
-- Playoff inter-confederaciones — 2 cupos
-- ═══════════════════════════════════════════════════════════════════════════

('00000000-0000-0000-0001-000000000047', 'República Dominicana', 'DOM', 'DOM', 'CONCACAF', 110, 'https://flagcdn.com/do.svg', TRUE),
('00000000-0000-0000-0001-000000000048', 'Indonesia',       'IDN', 'IDN', 'AFC',       130, 'https://flagcdn.com/id.svg', TRUE)

ON CONFLICT (id) DO UPDATE SET
    name               = EXCLUDED.name,
    short_name         = EXCLUDED.short_name,
    country_code       = EXCLUDED.country_code,
    confederation      = EXCLUDED.confederation,
    fifa_ranking_date  = EXCLUDED.fifa_ranking_date,
    flag_url           = EXCLUDED.flag_url,
    qualified          = EXCLUDED.qualified;

-- ── Verificación ──────────────────────────────────────────────────────────────
DO $$
DECLARE
    total      INT;
    uefa_c     INT;
    conmebol_c INT;
    concacaf_c INT;
    caf_c      INT;
    afc_c      INT;
    ofc_c      INT;
BEGIN
    SELECT COUNT(*)                                          INTO total      FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'UEFA')    INTO uefa_c     FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'CONMEBOL')INTO conmebol_c FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'CONCACAF')INTO concacaf_c FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'CAF')     INTO caf_c      FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'AFC')     INTO afc_c      FROM teams WHERE qualified = TRUE;
    SELECT COUNT(*) FILTER (WHERE confederation = 'OFC')     INTO ofc_c      FROM teams WHERE qualified = TRUE;

    RAISE NOTICE '=== Verificación seed_teams ===';
    RAISE NOTICE 'Total equipos clasificados : %  (esperado: 48)', total;
    RAISE NOTICE 'UEFA                        : %  (esperado: 16)', uefa_c;
    RAISE NOTICE 'CONMEBOL                    : %  (esperado: 6)',  conmebol_c;
    RAISE NOTICE 'CONCACAF                    : %  (esperado: 6)',  concacaf_c;
    RAISE NOTICE 'CAF                         : %  (esperado: 9)',  caf_c;
    RAISE NOTICE 'AFC                         : %  (esperado: 8)',  afc_c;
    RAISE NOTICE 'OFC                         : %  (esperado: 1)',  ofc_c;

    IF total != 48 THEN
        RAISE EXCEPTION 'Error: se esperaban 48 equipos, se insertaron %', total;
    END IF;

    RAISE NOTICE '✅ seed_teams completado correctamente.';
END $$;
