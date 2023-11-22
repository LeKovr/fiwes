# fiwes - File (Image) WEb Storage

[![GoDoc][gd1]][gd2]
 [![codecov][cc1]][cc2]
 [![GoCard][gc1]][gc2]
 [![GitHub Release][gr1]][gr2]
 [![Docker Image][di1]][di2]
 [![LoC][loc1]][loc2]
 [![GitHub license][gl1]][gl2]

[cc1]: https://codecov.io/gh/LeKovr/fiwes/branch/master/graph/badge.svg
[cc2]: https://codecov.io/gh/LeKovr/fiwes
[gd1]: https://godoc.org/github.com/LeKovr/fiwes?status.svg
[gd2]: https://godoc.org/github.com/LeKovr/fiwes
[gc1]: https://goreportcard.com/badge/github.com/LeKovr/fiwes
[gc2]: https://goreportcard.com/report/github.com/LeKovr/fiwes
[gr1]: https://img.shields.io/github/release/LeKovr/fiwes.svg
[gr2]: https://github.com/LeKovr/fiwes/releases
[di1]: https://img.shields.io/badge/docker-lekovr/fiwes-blue.svg
[di2]: https://hub.docker.com/r/lekovr/fiwes/
[loc1]: https://raw.githubusercontent.com/LeKovr/fiwes/master/.loc.svg?sanitize=true "Lines of Code"
[loc2]: https://github.com/LeKovr/fiwes/blob/master/LOC.md
[gl1]: https://img.shields.io/github/license/LeKovr/fiwes.svg
[gl2]: https://github.com/LeKovr/fiwes/blob/master/LICENSE

## Назначение

Приложение, которое принимает по http изображения, сохраняет их и делает превью 100х100 пикселей.

Входные данные: 
1. multipart/form-data
2. строка base64 в JSON
3. ссылка на изображение из сети как GET параметр

### Дополнения

1. test coverage >= 70%
2. поддержка деплоя командой `docker-compose up`

## Особенности реализации

* Т.к. список форматов изображений не задан (может, .svg?), принимаем, что список форматов аналогичен списку поддерживаемых выбранным пакетом ресайза ([imaging](https://github.com/disintegration/imaging))
* Т.к. цель - прием изображений, то при получении файла, который не является изображением (т.е. пакет не может выполнить ресайз), возвращается статус 415 (UnsupportedMediaType)
* В случаях, когда запрос не в JSON, сервер отвечает редиректом на превью. Для GET тоже, чтобы рефреш не повторял скачивание. По redirect url можно получить id изображения, отрезав префикс (заменив `/preview/` на `/img/`)
* Статус ошибки должен соответствовать некоторому стандарту, использованы предварительные варианты

## Архитектура

* [fiwes](https://github.com/LeKovr/fiwes) - Вебсервер на основе [gin-gonic](http://github.com/gin-gonic/gin)
* [upload](https://godoc.org/github.com/LeKovr/fiwes/upload) - прием и сохранение файла, создание preview с помощью [imaging](https://github.com/disintegration/imaging)
* [ginupload](https://godoc.org/github.com/LeKovr/fiwes/ginupload) - привязка upload к [gin-gonic](http://github.com/gin-gonic/gin)

## Деплой

```
wget https://raw.githubusercontent.com/LeKovr/fiwes/master/docker-compose.yml
docker-compose up
```
После запуска сервис доступен по адресу http://localhost:8080/.

### Локальная сборка

```
git clone https://github.com/LeKovr/fiwes.git
cd fiwes
make up
```

## Опции

Приложение поддерживает параметры конфигурации:
```
$ ./fiwes -h
fiwes 0.0-dev. File web storage server
Usage:
  fiwes [OPTIONS]

Application Options:
      --http_addr=          Http listen address (default: localhost:8080)
      --upload_limit=       Upload size limit (Mb) (default: 8)
      --html                Show html index page

Image upload Options:
      --img.download_limit= External image size limit (Mb) (default: 8)
      --img.dir=            Image upload destination (default: data/img)
      --img.preview_dir=    Preview image destination (default: data/preview)
      --img.preview_width=  Preview image width (default: 100)
      --img.preview_heigth= Preview image heigth (default: 100)
      --img.random_name     Do not keep uploaded image filename
      --img.path=           Image URL path (default: /img)
      --img.upload_path=    Image upload URL path (default: /upload)
      --img.preview_path=   Preview image URL path (default: /preview)

Help Options:
  -h, --help                Show this help message
```

## Docker

Образ docker создается "from scratch" и для запуска контейнера используются следующие файлы хост-системы:
* /etc/timezone, /etc/localtime - чтобы время в логах соответствовало серверному

Файл /etc/mime.types копируется в контейнер при сборке, для задания соответствий Content-Type расширениям (см [mime](https://golang.org/pkg/mime/#TypeByExtension)).

Все операции с docker производятся через контейнер docker-compose.

Приложение запускается в контейнере под пользователем nobody:nogroup и сохраняет файлы в `./var/data`. Чтобы создание файлов было доступно, перед стартом контейнера выполняется команда `mkdir -p -m 777 var/data/{img,preview}`.

## Использование

Основные операции, которые можно проделать с исходным кодом проекта, собраны в [Makefile](Makefile). Для получения справки по доступным операциям используется команда:
```
$ make
##
## Available make targets
##
## Sources
    run                Run from sources
    build-all          Build app with checks
    build              Build app
    build-standalone   Build app used in docker from scratch
    gen                Generate mocks
    fmt                Format go sources
    vet                Run vet
    lint               Run linter
    lint-more          Run more linters
    cov                Run tests and fill coverage.out
    cov-html           Open coverage report in browser
    cov-clean          Clean coverage report
## Docker
    up                 Start service in container
    down               Stop service
    build-docker       Build docker image
    clean-docker       Remove docker image & temp files
    dc                 run docker-compose
## Misc
    cloc               Count lines of code (including tests) and update LOC.md
    help               List Makefile targets

```

## Статусы ответа сервера

### 200. OK
* возвращается вместе с ответом в JSON при успешной загрузке изображения в base64

### 302. Found
* Редирект на превью, возвращается при загрузке изображения методом POST в "multipart/form-data"
* Редирект на превью, возвращается при загрузке изображения методом GET

### 400. BadRequest
* данные не соответствуют формату "multipart/form-data"
* JSON не соответствует структуре `{"name": .., "data":..}`
* в форме не передано поле "file" в единственном числе
* строка в base64 Не соответствует формату

### 415. UnsupportedMediaType
* Загруженный файл не может быть обработан как изображение
* Не удалось определить расширение файла по переданному Content-Type

### 500. InternalServerError
* Ошибка на стороне сервиса, подробности записаны в журнал

### 503. ServiceUnavailable
* Ошибка загрузки изображения по URL
* Статус ответа загрузки изображения по URL != 200

## См. также

* вебсервер - http://github.com/gin-gonic/gin
* ресайз
  * https://github.com/disintegration/imaging
  * https://github.com/nfnt/resize
  * https://github.com/anthonynsimon/bild
  * https://github.com/disintegration/gift
* аналоги
  * https://github.com/h2non/imaginary
  * https://github.com/aldor007/mort
  * https://github.com/thoas/picfit

### Варианты развития проекта

* [ ] add example?
* [ ] tests via docker?
* [ ] main.go: leave main() only and add `//+build !test`?
* [ ] create preview as symlink if image size is 100x100?
* [ ] rate limit
* [ ] HTTP Range, Conditional, Options requests
* [ ] Compression, for incoming base64 atleast

## License

The MIT License (MIT), see [LICENSE](LICENSE).

Copyright (c) 2019 Aleksei Kovrizhkin <lekovr+fiwes@gmail.com>
