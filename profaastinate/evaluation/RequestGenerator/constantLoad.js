import http from 'k6/http';
import { rate } from 'k6';

export let options = {
  stages: [
    { duration: '5m', target: 5 }, // 5 requests per second for 5 minutes
  ],
};

export default function () {
    let headers = {
        'x-nuclio-function-name': 'check',
        'x-nuclio-function-namespace': 'nuclio',
        'x-nuclio-async': 'true',
        'x-nuclio-async-deadline': '30000',
      };
    
      // Send a GET request
      let res = http.get('http://localhost:8070/api/function_invocations', { headers: headers });
      check(res, {"Response Status is 204": (r) => r.status == 204})
}