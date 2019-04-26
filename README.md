# imgserv

## Задача

Приложение, которое принимает по http изображения, сохраняет их и делает превью 100х100 пикселей

Входные данные: 
1. multipart/form-data
2. строка base64 в JSON
3. ссылка на изображение из сети как GET параметр

* Приложение должно быть покрыто тестами не менее чем на 70%.
* Приложение должно быть упаковано в Docker и разворачиваться с помощью одной команды docker-compose up.

## Архитектура

* cmd/imgserv - Вебсервер
* upload - прием и сохранение файла
* resize - ресайз
* preview - показ

## Зависимости

* вебсервер - http://github.com/gin-gonic/gin
* ресайз - 
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

TODO

* [x] refactoring
* [x] html
* [x] js (show preview?)
* [x] drop unsupported media
* [ ] tests
* [ ] docs

если оригинальный файл 100x100 - делать симлинк 

## Уточнения ТЗ

* не задан список форматов изображений (может, .svg?), это нужно при выборе пакета ресайза. Без этого уточнения принимает, что список форматов аналогичен списку выбранного пакета ресайза
* возвращаем только изображение (размер заранее известен), не потоки