async function fetchAllPhotos() {
    return await fetch("/graphql?query={queryImage(){objectPath}}")
	.then(resp => resp.json())
	.then(resp => resp.data.queryImage)
	.catch(error => {
	    alert(error);
	    return [];
	});
    
}

async function initGallery(gallery_elem_id) {
    imgs = await fetchAllPhotos();
    console.log(imgs);
    img_list = "<ul>";
    for (img of imgs) {
	img_list += "<li><a href='/photo/" + img.id + "'><img style='max-width: 300px' src='http://localhost:9000/" + img.objectPath + "' /></a></li>";
    }
    img_list += "</ul>";
    console.log(img_list);
    document.getElementById(gallery_elem_id).innerHTML = img_list;
}
