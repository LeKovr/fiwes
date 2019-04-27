# fiwes - FIle (Image) WEb Storage

## Назначение

Приложение, которое принимает по http изображения, сохраняет их и делает превью 100х100 пикселей

Входные данные: 
1. multipart/form-data
2. строка base64 в JSON
3. ссылка на изображение из сети как GET параметр

* Приложение должно быть покрыто тестами не менее чем на 70%.
* Приложение должно быть упаковано в Docker и разворачиваться с помощью одной команды docker-compose up.

## Архитектура

* fiwes - Вебсервер на основе [gin-gonic](http://github.com/gin-gonic/gin)
* upload - прием и сохранение файла, создание preview с помощью [imaging](https://github.com/disintegration/imaging)
* ginupload - привязка upload к [gin-gonic](http://github.com/gin-gonic/gin)

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
      --img.random_name     Don't keep uploaded image filename
      --img.path=           Image URL path (default: /img)
      --img.upload_path=    Image upload URL path (default: /upload)
      --img.preview_path=   Preview image URL path (default: /preview)

Help Options:
  -h, --help                Show this help message
```

## Docker

Образ docker создается "from scratch" и для запуска контейнера используются следующие файлы хост-системы:
* /etc/timezone, /etc/localtime - чтобы время в логах соответствовало серверному
* /etc/mime.types - для задания соответствий Content-Type расширениям (см [mime](https://golang.org/pkg/mime/#TypeByExtension))

Все операции с docker производятся через контейнер docker-compose

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
    test-docker        Test project via docker
    dc                 run docker-compose
## Misc
    cloc               Count lines of code (including tests) and update LOC.md
    help               List Makefile targets

```


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

## Что учесть

Вариант ответа - редирект на адрес картинки.
Для GET тоже - чтобы рефреш не повторял скачивание
по redirect url можно получить id изображения, отрезав префикс

Предусмотреть простоту замены библиотеки ресайза

### TODO

* [x] refactoring
* [x] html
* [x] js (show preview?)
* [x] drop unsupported media
* [x] docker
* [ ] tests
* [ ] docs
* [ ] если оригинальный файл 100x100 - делать симлинк 
* [ ] посмотреть аналоги

## Уточнения ТЗ

* не задан список форматов изображений (может, .svg?), это нужно при выборе пакета ресайза. Без этого уточнения принимает, что список форматов аналогичен списку выбранного пакета ресайза
* возвращаем только изображение (размер заранее известен), не потоки

