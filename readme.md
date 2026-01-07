The goal of this project is to make simple software which syncs markdown notes between my android phone and my computer.

I have tried syncthing in combination with obsidian.
The problem is syncthing's conflict resolution system is not super easy / intuitive.
I'd like to be able to jump to my computer if there is a conflict and resolve it there. 
Ideally there are not conflicts too often as syncs should happen quickly.
I like the idea of git for conflict resolution but for the actual syncing I think git might be too slow. 

Primarily this thing should be easy to use and set up on android (syncthing is not, although it is a joy to set up on linux)

I think a golang service would make sense for this project.
I'd like to avoid doing android development. A progressive web app would be fine.
Then I should be able to host it on whatever server I like.

Any Ideas for how to get started?
