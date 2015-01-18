angular.module('starter.controllers', [])

.controller('ChatsCtrl', function($scope, Chats) {
  $scope.chats = Chats.query();
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
