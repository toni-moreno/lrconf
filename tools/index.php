<?php
#var_dump($_FILES);
#var_dump($_REQUEST);
$nodeid=$_REQUEST["nodeid"];
$checkid=$_REQUEST["checkid"];
//if they DID upload a file...
foreach ( $_FILES as $prefijo => $file) {


   if($file['name'])
   {
        echo "Uploading file : ".$file['name']."\n";
        $valid_file = true;
        $message= "OK";

        //if no errors...
        if(!$file['error'])
        {
                //now is the time to modify the future file name and validate the file
                $new_file_name = strtolower($file['name']); //rename file
                if($file['size'] > (102400000)) //can't be larger than 1 MB
                {
                        $valid_file = false;
                        $message = 'Oops!  Your file\'s size is to large.';
                }

                //if the file has passed the test
                if($valid_file)
                {
                        $dir_path=$_SERVER["DOCUMENT_ROOT"].'/nodes/'.$nodeid.'/'.$checkid.'/uploads/';
                        if (!file_exists($dir_path)) {
                                mkdir($dir_path,0755,true);
                        }
                        $now= date('Y-m-d_H:i:s');
                        $file_name=$new_file_name.$now;
                        $full_path=$dir_path.$file_name;
                        //move it to where we want it to be
                        move_uploaded_file($file['tmp_name'], $full_path);
                        $message = 'Congratulations!  Your file was uploaded OK.';
                }
        }
        //if there is an error...
        else
        {
                //set that to be the returned message
                $message = 'Ooops!  Your upload triggered the following error:  '.$file['error'];
        }
    }
    echo $message."\n";
/*you get the following information for each file:
$_FILES['field_name']['name']
$_FILES['field_name']['size']
$_FILES['field_name']['type']
$_FILES['field_name']['tmp_name']*/
}
?>
