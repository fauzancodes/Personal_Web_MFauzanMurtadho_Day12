function sendMessage(){
  let name = document.getElementById('name').value;
  let email = document.getElementById('email').value;
  let phoneNumber = document.getElementById('phone-number').value;
  let subject = document.getElementById('subject').value;
  let message = document.getElementById('message').value;

  if (name != "" && email != "" && phoneNumber != "" && subject != "" && message != "") {
    let clientMessage = {
      name,
      email,
      phoneNumber,
      subject,
      message
    };
  
    const defaultEmail = "mfauzan.murtadho@gmail.com";
  
    let mailTo = document.createElement('a');
    mailTo.href = `mailto:${defaultEmail}?subject=${clientMessage.subject}&body=Greetings! My name is ${clientMessage.name}, ${clientMessage.message}, You can contact me at ${clientMessage.phoneNumber}.`;
    mailTo.target = "_blank";
    mailTo.click();
  }
}