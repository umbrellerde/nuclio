import http from 'k6/http';
import { rate, check } from 'k6';

export const options = {
    scenarios: {
      constant_request_rate: {
        executor: 'constant-arrival-rate',
        rate: 1,
        timeUnit: '1s',
        duration: '15m',
        preAllocatedVUs: 20,
        maxVUs: 100,
      },
    },
  };
export default function () {
    let headers = {
        'x-nuclio-function-name': 'ocr',
        'x-nuclio-function-namespace': 'nuclio',
      };
    
      // Send a GET request
      let res = http.get('http://localhost:8070/api/function_invocations', { headers: headers });
      check(res, {"Response Status is 204": (r) => r.status == 200})
}