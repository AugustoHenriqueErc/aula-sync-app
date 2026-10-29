CREATE TABLE conteudos (
    id INTEGER PRIMARY KEY,
    nome TEXT NOT NULL,
    nome_normalizado TEXT NOT NULL UNIQUE CHECK(length(nome_normalizado) > 0)
) STRICT;
