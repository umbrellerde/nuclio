import http from 'k6/http';
import { rate, check } from 'k6';

export const options = {
    scenarios: {
      constant_request_rate: {
        executor: 'constant-arrival-rate',
        rate: 1,
        timeUnit: '1s',
        duration: '30m',
        preAllocatedVUs: 20,
        maxVUs: 100,
      },
    },
  };
export default function () {
    let headers = {
        'x-nuclio-function-name': 'urgentcheck',
        'x-nuclio-function-namespace': 'nuclio',
      };
    
      const startTime = Date.now();
      // Send a GET request
      let res = http.get('http://localhost:8070/api/function_invocations', { headers: headers });
      console.log("K6MAGICSTRING" + startTime + ", " + res.timings.duration + "K7MAGICSTRING")
      check(res, {"Response Status is 204": (r) => r.status == 200})
}