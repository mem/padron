var padronApp = angular.module('padronApp', ['ui.bootstrap'])

// angular.module('myModule', ['ui.bootstrap']);

padronApp.controller('PadronCtrl', function ($scope, $http) {
  $scope.search = function() {
    var cedula = this.cedula.trim();
    if (cedula.match(/^\d+$/)) {
      switch (cedula.length) {
      case 7:
        // PMMMNNN => P0MMM0NNNN
        cedula = cedula.substring(0, 1)
                 + "0" + cedula.substring(1, 4)
                 + "0" + cedula.substring(4, 8);
        break;
      case 8:
        // PMMMNNNN => P0MMMNNNN
        cedula = cedula.substring(0, 1)
                 + "0" + cedula.substring(1, 4)
                 + cedula.substring(4, 8);
        break;
      case 9:
        // PMMMMNNNN, nothing to do!
        break;
      case 10:
        // 0PMMMMNNNN => PMMMMNNNN
        cedula = cedula.substring(1);
        break;
      }
    }
    $http.get('persona/' + cedula).success(function(data) {
      $scope.personas = [ data ];
      $scope.found = 1;
    }).error(function() {
      $scope.personas = [];
      $scope.found = 0;
    });
  };
});
