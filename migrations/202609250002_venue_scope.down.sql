-- Forward-only correction: the previous default violates its own CHECK.
-- Application rollback can retain the corrected schema. Do not restore an
-- invalid default or silently weaken validation.
DO $$
BEGIN
    RAISE EXCEPTION 'Venue scope correction is forward-only; retain the corrected schema on application rollback';
END
$$;
