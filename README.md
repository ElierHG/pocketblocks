![Static Badge](https://img.shields.io/badge/Project_State:-Under_Active_Development-green)

<img src="client/packages/openblocks/src/assets/images/logo-with-name.svg" width="280" alt="Logo">

Tradução deste arquivo em Português disponível [neste link](README.pt-br.md).

## What is PocketBlocks?

**Openblocks + PocketBase = PocketBlocks.**

PocketBlocks is a integration between Openblocks and PocketBase.

Traditionally, building an internal app requires complex frontend and backend interactions with hundreds and thousands lines of code, not to mention work on packaging, integration and deployment. PocketBlocks significantly reduces the work you need to do to build an app.

In PocketBlocks, all you need to do is drag and drop pre-built or self-customized components onto the What-You-See-Is-What-You-Get (WYSIWYG) canvas, PocketBlocks helps you build an app quickly and focus on business logic.

## Why choose PocketBlocks?

- **Open source**: Makes your ideas more feasible.
- **High scalability**: Allows to execute JavaScript almost anywhere you would like to customize your business processes and UI components.
- **Clean design**: Follows the principles of Ant Design and supports display on screens of different sizes. We have a number of UI components, based on which you can freely build dashboard, admin panel, and content management system (CMS).

## How to build an app in PocketBlocks?

Building an internal app basically takes 4 steps:

1. Quickly connect to your Pocketbase API using its SDK.
2. Use pre-built or user-customized UI components to build your app UI.
3. Set up [event handlers](docs/en/build-apps/event-handlers.md) to trigger javascript functions, control components or other actions in reaction to user interactions.
4. Preview and share your app with others.

## Connecting to Microsoft SQL Server

PocketBlocks can also connect to external Microsoft SQL Server instances. The
package `server/mssql` provides helper functions to open a `database/sql`
connection using environment variables:

```
MSSQL_HOST     - server hostname
MSSQL_PORT     - server port (defaults to 1433)
MSSQL_USER     - database user
MSSQL_PASSWORD - user password
MSSQL_DATABASE - default database name
```

```go
import "github.com/pedrozadotdev/pocketblocks/server/mssql"

db, err := mssql.Open(mssql.ConfigFromEnv())
if err != nil {
    // handle error
}
defer db.Close()
```

Administrators can create additional SQL Server connections from the dashboard.
These connections are stored in PocketBlocks and can be selected when building
queries in the app editor.

## License

AGPL3
