

 



  //modifying user data
  $(".btn").click(function() {

    
    var obj = {};
    obj.name = $('form input:text').text();
    obj.password = $('form input:password').text();
    
	url = '/admin'

    ajax_post(url,obj); 
  });


  //PUT Ajax calls
  function ajax_put(url,obj){
    $.ajax({
      url: url,
      type: 'POST',
      data: obj,
      success: function(result) {
          console.log(result);
	  //we can do some error checking here, as in ajax was good but DB was not a success
	  if(result.responseText != "success"){
            $("#errors").text("Something went wrong. User was not updated.")
	  }
      },
      error: function(result) {
          //general div to handle error messages
          $("#errors").text(result.responseText);
          //if martini binding recognizes a validation error, this is where you can decipher the JSON and properly display that shit
          // {"overall":{},"fields":{"Title":"Required","title":"Title cannot be empty"}}
          // JSON.parse(result.responseText).fields.title
      }
    });
  }


