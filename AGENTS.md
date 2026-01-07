The goal of this project is to make simple, self-hostable solution which syncs markdown notes between my desktop and android phone (progressive web app).
If there is a sync conflict I'd like to be able to jump to my computer and resolve it there (no need to resolve anything on the phone).
Ideally there are not conflicts too often as syncs should happen quickly.
I like the idea of git for conflict resolution but for the actual syncing I think git might be too slow. 
Primarily this thing should be easy to use and set up on android (syncthing is not, although it is a joy to set up on linux)

Build software to the user's specification.
When possible keep with the design patterns and coding standards of the project (unless you intend to improve the standard).
This project primarily uses server-side-rendered approach and `htmx` to build websites with the go programming language.
However, it is still possible to write Typescript in `web/index.ts` when a trip to the server doesn't make sense.
Using `templ` typescript functions can be called like so ```templ @templ.JSFuncCall("funcName", "argument1", "argument2") ```

The project primarily uses
- `golang/go`
- `a-h/templ`
- `tailwindlabs/tailwindcss`
- `saadeghi/daisyui`
- `oven-sh/bun`
- `bigskysoftware/htmx`
the project also uses `go-task/task` for building the project; similar to gnu make you can use `task build` to build the project

Feel free to do some refactoring to keep the code clean and concise.
Also if you deprecate code, then remove that code at the end of the change.
Do not leave unused functions and do not export functions which are not used outside that package.

After implementing the requested changes:

1. Run the build command: `task`
2. If there are any build errors, analyze them and fix the issues then re-run `task`
