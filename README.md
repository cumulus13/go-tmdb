# Go-TMDB

```basb
[36;1m
  🎬 moviedb — Movie & TV CLI powered by TMDb[0m
[2m  Data: The Movie Database (themoviedb.org)  |  Zero external dependencies[0m

[1mSETUP:[0m
  Windows:  set TMDB_API_KEY=your_key_here
  Linux:    export TMDB_API_KEY=your_key_here
  Get a free key at: https://www.themoviedb.org/settings/api

[1mCOMMANDS:[0m
  moviedb search    <query> [-t movie|tv|person] [-l N] [-y year]
  moviedb movie     <id>    [--export json|yaml|toml] [--lang xx-XX] [--region XX]
  moviedb tv        <id>    [--export json|yaml|toml] [--lang xx-XX] [--region XX]
  moviedb season    <tv_id> <season_num> [--export json|yaml|toml]
  moviedb person    <id>    [--export json|yaml|toml]
  moviedb images    <movie|tv|person> <id>  [--type poster|backdrop|logo|profile]
                            [--size w500]   [--export json|yaml|toml|csv]
  moviedb download  <movie|tv|person> <id>  [--type poster|backdrop|logo|all]
                            [--size w500]   [--dir ./my_images] [--limit N]
  moviedb videos    <movie|tv> <id>  [--export json|yaml|toml]
  moviedb reviews   <movie|tv> <id>  [--page N]
  moviedb trending  [-t movie|tv|all] [-w day|week] [--export json|yaml|toml]

[1mEXAMPLES:[0m
  moviedb search "Dune" -t movie -l 10
  moviedb movie 693134                        (Dune Part Two)
  moviedb movie 693134 --export json
  moviedb movie 693134 --export yaml --out dune.yaml
  moviedb movie 693134 --lang id-ID           (Indonesian)
  moviedb movie 693134 --region ID            (watch providers for Indonesia)
  moviedb tv 1396 --export toml              (Breaking Bad)
  moviedb season 1396 1 --export json
  moviedb person 6193 --export yaml          (Leonardo DiCaprio)
  moviedb images movie 693134 --type poster
  moviedb images movie 693134 --export csv   (all image URLs as CSV)
  moviedb images tv 1396 --type backdrop --size w1280
  moviedb download movie 693134 --type poster --size w500 --dir ./dune_images
  moviedb download tv 1396 --type all --limit 20
  moviedb videos movie 27205
  moviedb reviews movie 27205 --page 2
  moviedb trending -t tv -w day

[1mGLOBAL FLAGS:[0m
  --export   Output format: json, yaml, toml  (csv also for images command)
  --out      Output filename  (default: auto-generated)
  --lang     Language code, e.g. en-US, id-ID, ja-JP, fr-FR  (default: en-US)
  --region   Country code for watch providers, e.g. US, ID, GB  (default: US)

```

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)