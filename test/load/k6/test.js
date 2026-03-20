import {check} from 'k6';
import {SharedArray} from 'k6/data';
import {scenario} from 'k6/execution';
import http from 'k6/http';

const ids = new SharedArray('ids', function () {
    if (__ENV.IDS_FILE) {
        console.log("Reading", __ENV.IDS_FILE);
        let r = open(__ENV.IDS_FILE).split('\n');
        if (r.length > 1) {
            if (r[r.length - 1] === '') {
                r.pop()
            }
        }
        console.log("Done reading");

        return r;
    }
    console.log("ERROR", "no file specified");

    return null;
});

export const options = {
    scenarios: {
        'all-methods': {
            executor: 'shared-iterations',
            vus: __ENV.VUS || 100,
            iterations: ids.length,
            maxDuration: '15m',
        },
    },

};

export function get_host() {
    return __ENV.HOST_URL || 'http://localhost:8080';
}

const host_url = get_host();

export default function () {
    const id = ids[scenario.iterationInTest];
    const subject = __ENV.SUBJECT || 'test';
    request_reply(subject, id);
}

function request_reply(subject, payload) {
    const url = host_url + '/v1/request-reply?subject=' + subject;
    const headers = {
        headers: {
            'Content-Type': 'octet/stream',
        }
    }

    const res = http.request("POST", url, payload, headers);

    check(res, {
        'response code was OK': (res) => res.status >= 200 && res.status < 300,
        'response code was not 400': (res) => res.status !== 400,
    })
}
