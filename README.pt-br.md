![Static Badge](https://img.shields.io/badge/Estado_do_Projeto:-Sob_Desenvolvimento_Ativo-green)

<img src="client/packages/openblocks/src/assets/images/logo-with-name.svg" width="280" alt="Logo">

Translation of this file into English available at [this link](README.md).

## O que é PocketBlocks?

**Openblocks + PocketBase = PocketBlocks.**

PocketBlocks é uma integração entre Openblocks e PocketBase.

Tradicionalmente, construir um aplicativo interno requer interações complexas de front-end e back-end com centenas ou milhares de linhas de código, sem mencionar o trabalho de empacotamento, integração e implantação. PocketBlocks reduz significativamente o trabalho que você precisa fazer para construir um aplicativo.

No PocketBlocks, tudo o que você precisa fazer é arrastar e soltar componentes pré-construídos ou autopersonalizados na tela What-You-See-Is-What-You-Get (WYSIWYG). PocketBlocks ajuda você a construir um aplicativo rapidamente e se concentrar na logíca do negócio.

## Por que escolher PocketBlocks?

- **Código aberto**: Torna suas ideias mais viáveis.
- **Alta escalabilidade**: Permite executar JavaScript em praticamente qualquer lugar onde você gostaria de personalizar seus processos de negócios e componentes de UI.
- **Design limpo**: Segue os princípios do Ant Design e suporta exibição em telas de diferentes tamanhos. Temos vários componentes de UI, com base nos quais você pode construir livremente painel de controle, painel de administração e sistema de gerenciamento de conteúdo (CMS).

## Como construir um aplicativo no PocketBlocks?

Construir um aplicativo interno leva basicamente 4 passos:

1. Conecte-se rapidamente à sua API Pocketbase usando o SDK.
2. Use componentes de UI pré-construídos ou personalizados pelo usuário para construir a UI do seu aplicativo.
3. Configure [manipuladores de eventos](docs/pt-br/build-apps/event-handlers.md) para acionar funções javascript, controlar componentes ou outras ações em reação às interações do usuário.
4. Visualize e compartilhe seu aplicativo com outras pessoas.

## Conectando ao Microsoft SQL Server

É possível utilizar o pacote `server/mssql` para abrir conexões
`database/sql` com um servidor SQL Server. Os seguintes parâmetros podem ser
definidos via variáveis de ambiente:

```
MSSQL_HOST     - host do servidor
MSSQL_PORT     - porta (padrão 1433)
MSSQL_USER     - usuário
MSSQL_PASSWORD - senha
MSSQL_DATABASE - banco de dados padrão
```

```go
import "github.com/pedrozadotdev/pocketblocks/server/mssql"

db, err := mssql.Open(mssql.ConfigFromEnv())
if err != nil {
    // trate o erro
}
defer db.Close()
```

Administradores podem criar conexões adicionais de SQL Server pelo painel de
administração. Essas conexões ficam disponíveis para uso na criação de
consultas dentro do editor de aplicativos.

## Licença

AGPL3
