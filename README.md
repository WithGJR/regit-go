# ReGit - A Tiny Git-compatible Git Implementation 

ReGit is a tiny Git implementation written in Golang. It uses the same underlying file formats as Git. Therefore, all the changes made by ReGit can be checked by Git.

This project does not aim at implementing all the features of Git. It is just an experimental implementation for learning purpose.

## Available Commands

* `regit-go init`
* `regit-go add [file names]`
  * Ex: `regit-go add code/main.py README.md code/lib/util.py`
* `regit-go commit -m [message]`
  * `-m` option is required to supply.
  * Ex: `regit-go commit -m "init commit"`
* `regit-go checkout [path names]`
  * Ex: `regit-go checkout code/main.py code/lib/util.py`
* `regit-go branch [branch name]`
  * Ex: `regit-go branch develop`
