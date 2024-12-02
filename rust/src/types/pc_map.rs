#[derive(Debug)]
pub struct PcMap {
    from_ct: Vec<usize>,
    to_ct: Vec<usize>,
}

impl PcMap {
    pub fn new(size: usize) -> Self {
        Self {
            from_ct: vec![0; size],
            to_ct: vec![0; size],
        }
    }

    #[inline(always)]
    pub fn add_mapping(&mut self, orig: usize, converted: usize) {
        self.from_ct[orig] = converted;
        self.to_ct[converted] = orig;
    }

    #[inline(always)]
    pub fn to_ct(&self, converted: usize) -> usize {
        self.to_ct[converted]
    }

    #[inline(always)]
    pub fn to_converted(&self, orig: usize) -> usize {
        self.from_ct[orig]
    }
}
