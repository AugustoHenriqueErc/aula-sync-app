ALTER TABLE google_abas ADD COLUMN modelo INTEGER;
ALTER TABLE google_sincronizacoes ADD COLUMN conteudo_confirmado TEXT;
ALTER TABLE google_sincronizacoes ADD COLUMN conteudo_proposto TEXT;
ALTER TABLE google_sincronizacoes ADD COLUMN linha_proposta INTEGER;
PRAGMA user_version=5;
