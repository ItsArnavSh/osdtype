import { api } from './client';
import { ENDPOINTS } from './config';
import type { BugReport, BugReportInput, BugReportResponse } from './types';

export function submitBugReport(input: BugReportInput): Promise<BugReportResponse> {
	return api<BugReportResponse>(ENDPOINTS.BUGS, {
		method: 'POST',
		body: input,
		idempotencyKey: input.submission_id
	});
}

export async function myBugReports(): Promise<BugReport[]> {
	const res = await api<{ reports: BugReport[] }>(ENDPOINTS.BUGS_MINE);
	return res.reports;
}
