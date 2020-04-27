async function fetchAllPhotos() {
    return await fetch("/graphql?query={queryImage(){id objectPath}}")
	.then(resp => resp.json())
	.then(resp => resp.queryImage)
	.catch(error => {
	    alert(error);
	    return [];
	});
    
}

async function fetchImage(id) {
    return await fetch("/graphql?query={getImage(id: \"" + id + "\"){id objectPath}}")
	.then(resp => resp.json())
	.then(resp => resp.getImage)
	.catch(error => {
	    alert(error);
	    return {};
	});
    
}

async function fetchAllImageSegments(id) {
    return await fetch("/graphql?query={getImage(id: \"" + id + "\"){segments { upperLeftX upperLeftY lowerRightX lowerRightY }}}")    
	.then(resp => resp.json())
	.then(resp => resp.getImage.segments)
	.catch(error => {
	    alert(error);
	    return [];
	});
    
}

async function initGallery(gallery_elem_id) {
    imgs = await fetchAllPhotos();
    img_list = "<ul>";
    for (img of imgs) {
	img_list += "<li><a href='/photo?id=" + img.id + "'><img style='max-width: 300px' src='http://localhost:9000/" + img.objectPath + "' /></a></li>";
    }
    img_list += "</ul>";
    document.getElementById(gallery_elem_id).innerHTML = img_list;
}

async function initImg(img_elem_id, id) {
    img = await fetchImage(id);
    document.getElementById(img_elem_id).setAttribute("src", "http://localhost:9000/" + img.objectPath);
}

async function drawFaces(photo_elem_id, faces_canvas_elem_id, id){
    console.log(id);
    faces = await fetchAllImageSegments(id);
    
    var img = document.getElementById(photo_elem_id);
    var cnvs = document.getElementById(faces_canvas_elem_id);
    
    cnvs.style.position = "absolute";
    cnvs.style.left = img.offsetLeft + "px";
    cnvs.style.top = img.offsetTop + "px";
    cnvs.width = img.width;
    cnvs.height = img.height;
    
    var ctx = cnvs.getContext("2d");
    for (f of faces) {
	ctx.beginPath();
	ctx.lineWidth = 2;
	ctx.strokeStyle = 'red';    
	ctx.rect(f.upperLeftX, f.upperLeftY, f.lowerRightX - f.upperLeftX, f.lowerRightY - f.upperLeftY);
	ctx.stroke();
    }
}
