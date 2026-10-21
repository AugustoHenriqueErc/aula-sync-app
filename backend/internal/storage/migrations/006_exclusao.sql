-- O contador sobrevive à exclusão e evita reutilizar a chave externa de uma aula.
CREATE TABLE aula_sequencia (
 id INTEGER PRIMARY KEY CHECK(id=1),
 ultimo INTEGER NOT NULL CHECK(ultimo>=0)
) STRICT;
INSERT INTO aula_sequencia(id,ultimo) SELECT 1,coalesce(max(id),0) FROM aulas;
-- Intento de exclusão: fica apenas até confirmar o resultado remoto e apagar a aula.
CREATE TABLE aula_exclusoes (
 aula_id INTEGER PRIMARY KEY REFERENCES aulas(id),
 destino_id INTEGER REFERENCES google_destinos(id),
 remover_google INTEGER NOT NULL CHECK(remover_google IN (0,1))
) STRICT;
-- Posições usadas não serão reaproveitadas após limpeza ou timeout.
ALTER TABLE google_abas ADD COLUMN proxima_linha INTEGER NOT NULL DEFAULT 1 CHECK(proxima_linha>=1);
PRAGMA user_version=6;
