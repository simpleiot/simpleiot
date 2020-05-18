module Data.Data exposing (Data, empty)

import Data.Device as D
import Data.Group as G
import Data.User as U


empty : Data
empty =
    { groups = []
    , users = []
    , devices = []
    }


type alias Data =
    { groups : List G.Group
    , devices : List D.Device
    , users : List U.User
    }
