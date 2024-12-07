pub trait GetGenericStatic {
    type I<const STEPPABL: bool>;

    fn get_with_args<const STEPPABLE: bool>(
        t: &'static Self::I<true>,
        f: &'static Self::I<false>,
    ) -> &'static Self::I<STEPPABLE> {
        if STEPPABLE {
            // SAFETY:
            // STEPPABLE is true
            unsafe { std::mem::transmute::<&'static Self::I<true>, &'static Self::I<STEPPABLE>>(t) }
        } else {
            // SAFETY:
            // STEPPABLE is false
            unsafe {
                std::mem::transmute::<&'static Self::I<false>, &'static Self::I<STEPPABLE>>(f)
            }
        }
    }

    fn get<const STEPPABLE: bool>() -> &'static Self::I<STEPPABLE>;
}
