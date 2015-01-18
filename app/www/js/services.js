angular.module('starter.services', [
  'ngResource',
  'ngWebSocket'
])

.factory('Chats', function ($resource, $http, config) {
  var Chat = $resource(config.baseUrl + '/chats/:id',
    { id:'@id' });

  Chat.prototype.sendMsg = function (msg) {
    this.msgs = this.msgs || [];
    this.msgs.push(msg);
    console.log(msg);
    return $http.post(config.baseUrl + '/chats/'+this.id+'/messages', msg);
  };

  Chat.prototype.addMsg = function (msg) {
    this.msgs = this.msgs || [];
    console.log(msg);
    this.msgs.push(msg);
  };

  return Chat;
})

.factory('WS', function(config, Chats) {
  // Open a WebSocket connection
  //var dataStream = $websocket('wss://' + config.baseUrl.split('https://').split('%2C') + '/receive');
  //var dataStream = $websocket('ws://fuckit.ngrok.com/receive');
  var dataStream = new WebSocket('ws://fuckit.ngrok.com/receive');

  var chat;

  dataStream.onmessage = function (msg) {
    var parsed = JSON.parse(msg.data);
    chat.addMsg(parsed.data);
  };

  var methods = {
    get: function () {
      dataStream.send(JSON.stringify({ action: 'get' }));
    },
    setChat: function (c) {
      chat = c;
    }
  };

  return methods;
})
.controller('SomeController', function (MyData) {

  $scope.MyData = MyData;
})

/**
 * A simple example service that returns some data.
 */
.factory('Friends', function() {
  // Might use a resource here that returns a JSON array

  // Some fake testing data
  // Some fake testing data
  var friends = [{
    id: 0,
    name: 'Ben Sparrow',
    notes: 'Enjoys drawing things',
    face: 'https://pbs.twimg.com/profile_images/514549811765211136/9SgAuHeY.png'
  }, {
    id: 1,
    name: 'Max Lynx',
    notes: 'Odd obsession with everything',
    face: 'https://avatars3.githubusercontent.com/u/11214?v=3&s=460'
  }, {
    id: 2,
    name: 'Andrew Jostlen',
    notes: 'Wears a sweet leather Jacket. I\'m a bit jealous',
    face: 'https://pbs.twimg.com/profile_images/491274378181488640/Tti0fFVJ.jpeg'
  }, {
    id: 3,
    name: 'Adam Bradleyson',
    notes: 'I think he needs to buy a boat',
    face: 'https://pbs.twimg.com/profile_images/479090794058379264/84TKj_qa.jpeg'
  }, {
    id: 4,
    name: 'Perry Governor',
    notes: 'Just the nicest guy',
    face: 'https://pbs.twimg.com/profile_images/491995398135767040/ie2Z_V6e.jpeg'
  }];


  return {
    all: function() {
      return friends;
    },
    get: function(friendId) {
      // Simple index lookup
      return friends[friendId];
    }
  }
});
