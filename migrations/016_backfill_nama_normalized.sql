-- Migration 016: Backfill nama_normalized for existing entities
-- Uses simplified normalization (lowercase, strip punctuation/extra spaces).
-- Go normalizer also strips PT/CV/UD prefixes; re-normalized on entity update going forward.

UPDATE entities
SET nama_normalized = trim(regexp_replace(
    regexp_replace(
        regexp_replace(lower(trim(nama_utama)), '[.,\-''()"]', ' ', 'g'),
        '^(pt\.?|cv\.?|ud\.?|firma|koperasi|yayasan)\s+', '', 'g'
    ),
    '\s+(tbk\.?|persero)$', '', 'g'
))
WHERE nama_normalized IS NULL
  AND nama_utama IS NOT NULL
  AND trim(nama_utama) <> '';

UPDATE entities
SET nama_normalized = trim(regexp_replace(nama_normalized, '\s+', ' ', 'g'))
WHERE nama_normalized IS NOT NULL;
