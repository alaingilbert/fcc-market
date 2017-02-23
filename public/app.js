var module = angular.module('app', ['highcharts-ng', 'ngWebSocket']);

module.controller('AppController', function($scope, $websocket) {
    var ctrl = this;

    var loc = window.location;
    var uri = 'ws:';
    if (loc.protocol === 'https:') {
        uri = 'wss:';
    }
    uri += '//' + loc.host;
    uri += loc.pathname + 'ws';
    var ws = $websocket(uri);

    ws.onMessage(function(message) {
        var data = JSON.parse(message.data);
        if (data.Action == "init") {
            var stocks = data.Data;
            _.each(stocks, (v, k) => {
                ctrl.stockNames.push(k);
                ctrl.chartConfig.series.push({
                    name: k,
                    data: v.dataset.data,
                });
            });
        } else if (data.Action == "add") {
            var name = data.Data.dataset.dataset_code;
            ctrl.stockNames.push(name);
            ctrl.chartConfig.series.push({
                name: name,
                data: data.Data.dataset.data,
            });
        } else if (data.Action == "del") {
            var code = data.Data;
            ctrl.stockNames = _.without(ctrl.stockNames, code);
            _.remove(ctrl.chartConfig.series, (o) => { return o.name == code; })
        }
    });

    ctrl.stockNames = [];

    ctrl.btnRemoveStockClicked = function(name) {
        ws.send(JSON.stringify({action: "del", data: name}));
    };

    ctrl.btnSendClicked = function() {
        ws.send(JSON.stringify({action: "add", data: ctrl.code}));
        ctrl.code = "";
    };

    ctrl.chartConfig = {
        options: {
            rangeSelector: {
                selected: 4
            },
            yAxis: {
                plotLines: [{
                    value: 0,
                    width: 2,
                    color: "silver"
                }]
            },
            tooltip: {
                pointFormat: '<span style="color:{series.color}">{series.name}</span>: <b>{point.y}</b> ({point.change}%)<br/>',
                valueDecimals: 2
            },
            title: {
                text: "Stocks"
            },
            loading: !1,
            useHighStocks: !0,
        },
        series: [],

    };
});