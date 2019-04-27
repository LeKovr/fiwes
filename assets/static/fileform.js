//code from https://stackoverflow.com/a/23669825/5199825
function encodeImageFileAsURL() {
  var filesSelected = document.getElementById("inputFileToLoad").files;
  if (filesSelected.length > 0) {
    var fileToLoad = filesSelected[0];
    document.getElementById("inputDataName").value=fileToLoad.name;
    var fileReader = new FileReader();
    fileReader.onload = function(fileLoadedEvent) {
      var srcData = fileLoadedEvent.target.result; // <--- data: base64
      var newImage = document.createElement('img');
      newImage.src = srcData;
      document.getElementById("imgTest").innerHTML = newImage.outerHTML;
      document.getElementById("inputDataToSend").value=srcData;
      // console.log("Converted Base64 version is " + document.getElementById("imgTest").innerHTML);
    }
    fileReader.readAsDataURL(fileToLoad);
  }
}

// save file
function save(f, path) {
  var data = {
        name: f.elements["name"].value,
        data: f.elements["data"].value
      },
      div  = document.getElementById("imgTest"),
      xhr  = new XMLHttpRequest();
  div.innerHTML = '';
  disable_form(f, true);
  console.log('Call');
  xhr.open('POST', path, true);
  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.onreadystatechange = function() {
    if (xhr.readyState != 4) return;
      console.log('Done');
    if (xhr.status != 200) {
      console.log(xhr.status + ': ' + xhr.statusText);
      div.innerHTML = xhr.statusText;
    } else {
      console.log('Result: ' + xhr.responseText);
      rv = JSON.parse(xhr.responseText);
      var img = document.createElement('img'),
          a   = document.createElement('a');
      img.src = rv.preview;
      a.href = rv.file;
      a.appendChild(img);
      div.innerHTML = a.outerHTML;
    }
    disable_form(f, false);
  }
  xhr.send(JSON.stringify(data));
}

// code from https://gist.github.com/Peacegrove/5534309
function disable_form(form, state) {
  var elemTypes = ['input', 'textarea', 'button', 'select'];
  elemTypes.forEach(function callback(type) {
    var elems = form.getElementsByTagName(type);
    disable_elements(elems, state);
  });
}

// Disables a collection of form-elements.
function disable_elements(elements, state) {
  var length = elements.length;
  while(length--) {
    elements[length].disabled = state;
  }
}
