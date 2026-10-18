CREATE TABLE google_destinos (
 id INTEGER PRIMARY KEY,
 planilha_id TEXT NOT NULL UNIQUE CHECK(length(planilha_id)>0),
 criado_em TEXT NOT NULL
) STRICT;
CREATE TABLE google_configuracao (
 id INTEGER PRIMARY KEY CHECK(id=1),
 instalacao_id TEXT NOT NULL UNIQUE,
 destino_id INTEGER REFERENCES google_destinos(id),
 versao INTEGER NOT NULL DEFAULT 0 CHECK(versao>=0)
) STRICT;
INSERT INTO google_configuracao(id,instalacao_id) VALUES (1,lower(hex(randomblob(16))));
-- Vínculos serão preenchidos somente após inspeção real na etapa 11.
CREATE TABLE google_abas (
 destino_id INTEGER NOT NULL REFERENCES google_destinos(id),
 periodo TEXT NOT NULL CHECK(length(periodo)=7 AND periodo GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]' AND substr(periodo,1,4)>='0001' AND substr(periodo,6,2) BETWEEN '01' AND '12'),
 sheet_id INTEGER CHECK(sheet_id>=0),
 estado TEXT NOT NULL DEFAULT 'nao_verificada' CHECK(estado IN ('nao_verificada','ausente','vazia','incompativel','pronta')),
 verificada_em TEXT,
 PRIMARY KEY(destino_id,periodo),
 UNIQUE(destino_id,sheet_id),
 CHECK(estado!='pronta' OR sheet_id IS NOT NULL)
) STRICT;
CREATE TABLE google_sincronizacoes (
 aula_id INTEGER NOT NULL REFERENCES aulas(id),
 destino_id INTEGER NOT NULL,
 periodo TEXT NOT NULL,
 versao_enviada INTEGER CHECK(versao_enviada>0),
 estado TEXT NOT NULL DEFAULT 'aguardando_preparacao' CHECK(estado IN ('aguardando_preparacao','pendente','sincronizado','erro')),
 tentativas INTEGER NOT NULL DEFAULT 0 CHECK(tentativas>=0),
 ultima_tentativa TEXT,
 erro TEXT,
 PRIMARY KEY(aula_id,destino_id,periodo),
 FOREIGN KEY(destino_id,periodo) REFERENCES google_abas(destino_id,periodo),
 CHECK(estado!='sincronizado' OR versao_enviada IS NOT NULL)
) STRICT;
PRAGMA user_version=4;
