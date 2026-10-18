ALTER TABLE aulas ADD COLUMN versao INTEGER NOT NULL DEFAULT 1 CHECK(versao > 0);
ALTER TABLE aulas ADD COLUMN cancelada INTEGER NOT NULL DEFAULT 0 CHECK(cancelada IN (0,1));
-- Vazio identifica registros antigos cuja data de atualização não era conhecida.
ALTER TABLE aulas ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
CREATE TABLE aula_alteracoes (
 id INTEGER PRIMARY KEY,
 aula_id INTEGER NOT NULL REFERENCES aulas(id),
 versao_anterior INTEGER NOT NULL,
 operacao TEXT NOT NULL CHECK(operacao IN ('editar','cancelar','reativar')),
 registrada_em TEXT NOT NULL,
 estado_anterior TEXT NOT NULL CHECK(json_valid(estado_anterior)),
 UNIQUE(aula_id,versao_anterior)
) STRICT;
PRAGMA user_version = 3;
