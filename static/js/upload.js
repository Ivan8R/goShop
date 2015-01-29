
$("#file").change(function(){
var src=$("#file").val();
if(src!="")
{
formdata= new FormData();  // initialize formdata
var numfiles=this.files.length;  // number of files
var i, file, progress, size;
for(i=0;i<numfiles;i++)
{
file = this.files[i];
size = this.files[i].size;
name = this.files[i].name;
if (!!file.type.match(/image.*/))  // Verify image file or not
{
if((Math.round(size))<=(1024*1024)) //Limited size 1 MB
{
var reader = new FileReader();  // initialize filereader
reader.readAsDataURL(file);  // read image file to display before upload
$("#preview").show();
$('#preview').html("");
reader.onloadend = function(e){
var image = $('<img>').attr('src',e.target.result);
$(image).appendTo('#preview');
};
formdata.append("file[]", file);  // adding file to formdata
if(i==(numfiles-1))
{
$("#info").html("wait a moment to complete upload");
$.ajax({
    url: "upload.php",
    type: "POST",
    data: formdata,
    processData: false,
    contentType: false,
    success: function(res){
    if(res!="0")
    $("#info").html("Successfully Uploaded");
    else
    $("#info").html("Error in upload. Retry");
    }
    });
}
}
else
{
$("#info").html(name+"Size limit exceeded");
$("#preview").hide();
return;
}
}
else
{
$("#info").html(name+"Not image file");
$("#preview").hide();
return;
}
}
}
else
{
$("#info").html("Select an image file");
$("#preview").hide();
return;
}
return false;
});

$(".sendtoserver").click(function(e){
     alert("yesssssss");
});


