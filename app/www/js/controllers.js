angular.module('starter.controllers', [])

.controller('ChatsCtrl', function($scope, Chats) {
  $scope.chats = Chats.query(function () {
    for (var i=0; i < $scope.chats.length; i++) {
      for (var j=0; j < $scope.chats[i].participants.length; j++) {
        var unParsed = $scope.chats[i].participants[j];
        var participant= JSON.parse(unParsed);

        var display = '';
        if (participant.first_name && participant.last_name) {
          display = participant.first_name + ' ' + participant.last_name;
        } else if (participant.first_name) {
          display = participant.first_name;
        } else {
          display = participant.handle;
        }

        participant.display = display;

        $scope.chats[i].participants[j] = participant;
      }
    }
  });

  $scope.remove = function(chat) {
    Chats.remove(chat);
  }
})

.controller('ChatDetailCtrl', function($scope, $timeout, $stateParams,
  $ionicScrollDelegate, Chats) {

  var isIOS = ionic.Platform.isWebView && ionic.Platform.isIOS();

  $scope.myId = '12345';
  $scope.data = {};
  $scope.chat = Chats.get({id: $stateParams.chatId});

  $scope.sendMessage = function () {
    var d = new Date();
    d = d.toLocaleTimeString().replace(/:\d+ /, ' ');

    $scope.chat.$sendMsg({
      userId: 'TODO',
      text: $scope.data.message,
      time: d
    });

    delete $scope.data.message;
    $ionicScrollDelegate.scrollBottom(true);
  };

  $scope.inputUp = function () {
    if (isIOS) {
      $scope.data.keyboardHeight = 216;
    }
  };

  $scope.inputDown = function () {
    if (isIOS) {
      $scope.data.keyboardHeight = 0;
    }
    $ionicScrollDelegate.resize();
  };

  $scope.closeKeyboard = function () {
    // cordova.plugins.Keyboard.close();
  };
})

.controller('AccountCtrl', function($scope) {
  $scope.settings = {
    enableFriends: true
  };
});
