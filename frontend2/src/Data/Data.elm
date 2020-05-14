module Data.Data exposing (Data, empty)

import Data.Device as D
import Data.Org as O
import Data.User as U


empty : Data
empty =
    { orgs = []
    , users = []
    , devices = []
    }


type alias Data =
    { orgs : List O.Org
    , devices : List D.Device
    , users : List U.User
    }
