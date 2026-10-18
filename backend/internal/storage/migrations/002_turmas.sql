CREATE TABLE turmas (id INTEGER PRIMARY KEY, nome TEXT NOT NULL, nome_normalizado TEXT NOT NULL UNIQUE CHECK(length(nome_normalizado) > 0)) STRICT;
CREATE TABLE aulas_novas (
    turma_id INTEGER NOT NULL REFERENCES turmas(id),
    id INTEGER PRIMARY KEY CHECK (id > 0),
    data TEXT NOT NULL CHECK (
        length(data) = 10
        AND data GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
        AND data >= '0001-01-01'
        AND date(data, '+0 days') IS data
    ),
    inicio TEXT NOT NULL CHECK (
        length(inicio) = 5 AND inicio GLOB '[0-2][0-9]:[0-5][0-9]' AND inicio <= '23:59'
    ),
    fim TEXT NOT NULL CHECK (
        length(fim) = 5 AND fim GLOB '[0-2][0-9]:[0-5][0-9]' AND fim <= '23:59'
        AND fim > inicio
    ),
    turma TEXT NOT NULL CHECK (length(trim(turma, char(9, 10, 11, 12, 13, 32))) > 0),
    conteudo TEXT NOT NULL CHECK (length(trim(conteudo, char(9, 10, 11, 12, 13, 32))) > 0),
    observacoes TEXT NOT NULL DEFAULT '',
    intervalo_minutos INTEGER NOT NULL DEFAULT 0 CHECK (intervalo_minutos >= 0),
    duracao_minutos INTEGER NOT NULL CHECK (
        duracao_minutos > 0
        AND duracao_minutos = (
            CAST(substr(fim, 1, 2) AS INTEGER) * 60 + CAST(substr(fim, 4, 2) AS INTEGER)
            - CAST(substr(inicio, 1, 2) AS INTEGER) * 60 - CAST(substr(inicio, 4, 2) AS INTEGER)
            - intervalo_minutos
        )
    )
) STRICT;

