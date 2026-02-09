
---
---

### project description

- Pad syncs markdown notes between my desktop and android phone (progressive web app).
- It should be simple and, self-hostable.
- The core data should be the filesystem as the source of truth with metadata to complement.
- If there is a sync conflict the server should always win with the client taking the server data and an error alert.
- Syncs should happen quickly in order to avoid conflicts.
- The main interface to edit notes on desktop is Neovim and any changes to filesystem should immediately push to the PWA.
- Pad should be easy to use and set up on mobile devices.

### technologies

core tech stack includes:
- `golang/go`
- `a-h/templ`
- `tailwindlabs/tailwindcss`
- `saadeghi/daisyui`
- `oven-sh/bun`
- `bigskysoftware/htmx`
the project also uses `go-task/task` for building the project; similar to gnu make you can use `task build` to build the project

### solving problems

Build new software to the user's specification.
When possible keep with the design patterns and coding standards of the project.
If you can improve the coding standards, treat that as refactor & improve project wide.

choose a solution type like so:
1. try to use daisy ui, and hypertext solution first (some problems don't require a server)
    - all of the styling and hypertext is in `ui` directory and `web/input.css`
    - if you need to debug daisy ui you can inspect pieces of `build/css/tailwind-generated.min.css`
2. if that fails, go with simple HTMX / golang server-side-rendered solution (this happens often)
    - most golang handler logic is in the `handle` directory
    - the main HTML templating, templ, HTMX is in the `ui` directory
    - there is also a filesystem / data layer in the `notes` directory
3. finally if it really doesn't make sense to take a trip to the server write some simple typescript (least common)
    - entrypoint for Typescript is `web/index.ts`
    - Using `templ` typescript functions can be called like so ```templ @templ.JSFuncCall("funcName", "argument1", "argument2") ```

Feel free to do some refactoring to keep the code clean and concise.
Also if you deprecate code, then remove that code; do not leave unused functions.
Do not export functions which are not used outside that package.
Try to extract common logic to avoid duplicate logic which can lead to divergence bugs.

### after implementing

1. Run the build command: `task`
2. If there are any build errors, analyze them and fix the issues
3. then run the integration tests with `task test`
3. fix any broken tests and re-run `task test` until the build passes
