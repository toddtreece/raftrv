import http from 'k6/http';
import exec from 'k6/execution';
import { check } from 'k6';

export const options = {
  discardResponseBodies: true,
  scenarios: {
    contacts: {
      executor: 'shared-iterations',
      vus: 10,
      iterations: 50000,
      maxDuration: '30s',
    },
  },
};

export default function () {
  const iteration = exec.scenario.iterationInInstance;
  let node;
  switch(iteration % 3) {
    case 0:
      node = 1;
      break;
    case 1:
      node = 2;
      break;
    case 2:
      node = 3;
      break;
  }
  http.patch(`http://127.0.0.1:${node}2380/`, JSON.stringify({node, iteration}), {
    responseType: 'text',
  });
}
