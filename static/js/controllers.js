var padronApp = angular.module('padronApp', ['ui.bootstrap'])

// angular.module('myModule', ['ui.bootstrap']);

padronApp.controller('PadronCtrl', function ($scope, $http) {
  $scope.search = function() {
    $http.get('persona/' + this.cedula).success(function(data) {
      $scope.personas = [ data ];
      $scope.found = 1;
    }).error(function() {
      $scope.personas = [];
      $scope.found = 0;
    });
  };
});
